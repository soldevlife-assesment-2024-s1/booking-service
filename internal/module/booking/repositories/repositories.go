package repositories

import (
	"booking-service/config"
	"booking-service/internal/module/booking/models/entity"
	"booking-service/internal/module/booking/models/response"
	"booking-service/internal/pkg/errors"
	"booking-service/internal/pkg/scheduler"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	circuit "github.com/rubyist/circuitbreaker"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

type repositories struct {
	db              *sqlx.DB
	log             *otelzap.Logger
	httpClient      *circuit.HTTPClient
	cfg             *config.Config
	redisClient     *redis.Client
	clientScheduler *asynq.Client
	cb              *circuit.Breaker
}

// SubmitPayment implements Repositories.
func (r *repositories) SubmitPayment(ctx context.Context, bookingID string, amount float64, paymentMethod string, paymentDate time.Time) error {
	if !r.cb.Ready() {
		return errors.InternalServerError("error submit payment, 3rd party service is down")
	}
	url := fmt.Sprintf(r.cfg.PaymentService.Endpoint)
	payload := map[string]interface{}{
		"booking_id":     bookingID,
		"amount":         amount,
		"payment_method": paymentMethod,
		"payment_date":   paymentDate,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.InternalServerError("error submit payment")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return errors.InternalServerError("error submit payment")
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return errors.InternalServerError("error submit payment")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Submit payment failed: %d", resp.StatusCode))
		return errors.BadRequest("Submit payment failed")
	}

	return nil
}

// SetTaskScheduler implements Repositories.
func (r *repositories) SetTaskScheduler(ctx context.Context, expiredAt time.Duration, payload []byte) (taskID string, err error) {
	taskPaymentExpiredAt := asynq.NewTask(scheduler.TypeSetPaymentExpired, payload, asynq.MaxRetry(3), asynq.Timeout(expiredAt))

	taskInfo, err := r.clientScheduler.Enqueue(taskPaymentExpiredAt, asynq.ProcessIn(expiredAt))
	if err != nil {
		return "", errors.InternalServerError("error enqueue task payment expired")
	}

	return taskInfo.ID, nil
}

// DeleteTaskScheduler implements Repositories.
func (r *repositories) DeleteTaskScheduler(ctx context.Context, taskID string) error {
	if !r.cb.Ready() {
		return errors.InternalServerError("error delete task scheduler")
	}
	url := fmt.Sprintf("http://%s:%s/monitoring/api/queues/default/scheduled_tasks:batch_delete", r.cfg.SchedulerService.Host, r.cfg.SchedulerService.Port)
	payload := map[string]interface{}{
		"task_ids": []string{taskID},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.InternalServerError("error delete task scheduler")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return errors.InternalServerError("error delete task scheduler")
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return errors.InternalServerError("error delete task scheduler")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println(resp)
		r.log.Ctx(ctx).Error(fmt.Sprintf("Delete task scheduler failed: %d", resp.StatusCode))
		return errors.BadRequest("Delete task scheduler failed")
	}

	return nil
}

// FindBookingByID implements Repositories.
func (r *repositories) FindBookingByID(ctx context.Context, bookingID string) (entity.Booking, error) {
	query := `SELECT * FROM bookings WHERE id = $1`
	var booking entity.Booking
	err := r.db.GetContext(ctx, &booking, query, bookingID)
	if err == sql.ErrNoRows {
		return entity.Booking{}, nil
	}
	if err != nil {
		return entity.Booking{}, errors.InternalServerError("error find booking by booking id")
	}
	return booking, nil
}

// InquiryTicketAmount implements Repositories.
func (r *repositories) InquiryTicketAmount(ctx context.Context, ticketDetailID int64, totalTicket int) (float64, error) {
	if !r.cb.Ready() {
		return 0, errors.InternalServerError("error inquiry ticket amount")
	}
	url := fmt.Sprintf("http://%s:%s/api/private/ticket/inquiry?ticket_detail_id=%d&total_ticket=%d", r.cfg.TicketService.Host, r.cfg.TicketService.Port, ticketDetailID, totalTicket)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, errors.InternalServerError("error inquiry ticket amount")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, errors.InternalServerError("error inquiry ticket amount")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Inquiry ticket amount failed: %d", resp.StatusCode))
		return 0, errors.BadRequest("Inquiry ticket amount failed")
	}

	// parse response
	var respBase response.BaseResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&respBase); err != nil {
		return 0, err
	}

	respBase.Data = respBase.Data.(map[string]interface{})
	respData := response.InquiryTicketAmount{
		TotalTicket: int(respBase.Data.(map[string]interface{})["total_ticket"].(float64)),
		TotalAmount: respBase.Data.(map[string]interface{})["total_amount"].(float64),
	}

	return respData.TotalAmount, nil
}

// FindBookingByBookingID implements Repositories.
func (r *repositories) FindBookingByBookingID(ctx context.Context, bookingID string) (entity.Booking, error) {
	query := fmt.Sprintf(`SELECT * FROM bookings WHERE id = %s`, bookingID)
	var booking entity.Booking
	err := r.db.GetContext(ctx, &booking, query)
	if err == sql.ErrNoRows {
		return entity.Booking{}, nil
	}
	if err != nil {
		return entity.Booking{}, errors.InternalServerError("error find booking by booking id")
	}
	return booking, nil
}

// CheckStockTicket implements Repositories.
func (r *repositories) CheckStockTicket(ctx context.Context, ticketDetailID int64) (int64, error) {
	ticketIDString := fmt.Sprintf("%d", ticketDetailID)
	data, err := r.redisClient.Get(ctx, ticketIDString).Result()
	if err != nil {
		if !r.cb.Ready() {
			return 0, errors.InternalServerError("error check stock ticket")
		}
		// hit api ticket service to get stock ticket
		url := fmt.Sprintf("http://%s:%s/api/private/ticket/stock?ticket_detail_id=%d", r.cfg.TicketService.Host, r.cfg.TicketService.Port, ticketDetailID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, errors.InternalServerError("error get stock ticket")
		}
		resp, err := r.httpClient.Do(req)
		if err != nil {
			return 0, errors.InternalServerError("error get stock ticket")
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			r.log.Ctx(ctx).Error(fmt.Sprintf("Get stock ticket failed: %d", resp.StatusCode))
			return 0, errors.BadRequest("Get stock ticket failed")
		}

		// parse response
		var respBase response.BaseResponse

		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&respBase); err != nil {
			return 0, err
		}

		respBase.Data = respBase.Data.(map[string]interface{})
		data = fmt.Sprintf("%d", int64(respBase.Data.(map[string]interface{})["stock"].(float64)))

		// set stock ticket to redis

		_, err = r.redisClient.Set(ctx, ticketIDString, data, 0).Result()
		if err != nil {
			return 0, errors.InternalServerError("error set stock ticket")
		}
	}
	dataInt, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, errors.InternalServerError("error parse stock ticket")
	}
	return dataInt, nil
}

// DecrementStockTicket implements Repositories.
func (r *repositories) DecrementStockTicket(ctx context.Context, ticketDetailID int64) error {
	ticketIDString := fmt.Sprintf("%d", ticketDetailID)
	_, err := r.redisClient.Decr(ctx, ticketIDString).Result()
	if err != nil {
		return errors.InternalServerError("error decrement stock ticket")
	}
	return nil
}

// IncrementStockTicket implements Repositories.
func (r *repositories) IncrementStockTicket(ctx context.Context, ticketDetailID int64) error {
	ticketIDString := fmt.Sprintf("%d", ticketDetailID)
	_, err := r.redisClient.Incr(ctx, ticketIDString).Result()
	if err != nil {
		return errors.InternalServerError("error increment stock ticket")
	}
	return nil
}

// UpsertBooking implements Repositories.
func (r *repositories) UpsertBooking(ctx context.Context, booking *entity.Booking) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", errors.InternalServerError("error starting transaction")
	}

	// Lock the rows for update
	query := `SELECT * FROM bookings WHERE id = $1 FOR UPDATE`
	var existingBooking entity.Booking
	err = r.db.GetContext(ctx, &existingBooking, query, booking.ID)
	if err != nil && err != sql.ErrNoRows {
		err = tx.Rollback()
		if err != nil {
			return "", errors.InternalServerError("error rolling back transaction")
		}
		return "", errors.InternalServerError("error locking rows")
	}

	var ID string

	// Perform the upsert operation
	if err == sql.ErrNoRows {
		// Insert new booking
		queryInsert := fmt.Sprintf(`
			INSERT INTO bookings (user_id, ticket_detail_id, total_tickets, full_name, personal_id, booking_date) 
			VALUES (%d, %d, %d, '%s', '%s', '%s') RETURNING id
		`, booking.UserID, booking.TicketDetailID, booking.TotalTickets, booking.FullName, booking.PersonalID, booking.BookingDate.Format("2006-01-02 15:04:05"))

		err := tx.QueryRowContext(ctx,
			queryInsert).Scan(&ID)
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				return "", errors.InternalServerError("error rolling back transaction")
			}
			return "", errors.InternalServerError("error upserting booking")
		}
	} else {
		// Update existing booking
		queryUpdate := fmt.Sprintf(`
			UPDATE bookings
			SET user_id = %d, ticket_detail_id = %d, total_tickets = %d, full_name = '%s', personal_id = '%s', booking_date = '%s'
			WHERE id = '%s' RETURNING id
		`, booking.UserID, booking.TicketDetailID, booking.TotalTickets, booking.FullName, booking.PersonalID, booking.BookingDate.Format("2006-01-02 15:04:05"), booking.ID)
		err := tx.QueryRowContext(ctx, queryUpdate).Scan(&ID)
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				return "", errors.InternalServerError("error rolling back transaction")
			}
			return "", errors.InternalServerError("error upserting booking")
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", errors.InternalServerError("error committing transaction")
	}

	return ID, nil
}

// UpsertPayment implements Repositories.
func (r *repositories) UpsertPayment(ctx context.Context, payment *entity.Payment) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.InternalServerError("error starting transaction")
	}

	// Lock the rows for update
	query := `SELECT * FROM payments WHERE booking_id = $1 FOR UPDATE`
	var existingPayment entity.Payment
	err = r.db.GetContext(ctx, &existingPayment, query, payment.BookingID)
	if err != nil && err != sql.ErrNoRows {
		err = tx.Rollback()
		if err != nil {
			return errors.InternalServerError("error rolling back transaction")
		}
		return errors.InternalServerError("error locking rows")
	}

	// Perform the upsert operation
	if err == sql.ErrNoRows {
		// Insert new payment
		queryInsert := fmt.Sprintf(`
			INSERT INTO payments (booking_id, amount, currency, status, payment_method, payment_date, payment_expiration, task_id)
			VALUES ('%s', %f, '%s', '%s', '%s', '%s', '%s','%s')
		`, payment.BookingID.String(), payment.Amount, payment.Currency, payment.Status, payment.PaymentMethod, payment.PaymentDate.Format("2006-01-02 15:04:05"), payment.PaymentExpiration.Format("2006-01-02 15:04:05"), payment.TaskID)
		_, err := tx.ExecContext(ctx, queryInsert)
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				return errors.InternalServerError("error rolling back transaction")
			}
			return errors.InternalServerError("error upserting payment")
		}
	} else {
		// Update existing payment
		queryUpdate := fmt.Sprintf(`
			UPDATE payments
			SET amount = %f, currency = '%s', status = '%s', payment_method = '%s', payment_date = '%s', payment_expiration = '%s', task_id = '%s'
			WHERE booking_id = '%s'
		`, payment.Amount, payment.Currency, payment.Status, payment.PaymentMethod, payment.PaymentDate.Format("2006-01-02 15:04:05"), payment.PaymentExpiration.Format("2006-01-02 15:04:05"), payment.TaskID, payment.BookingID.String())
		_, err := tx.ExecContext(ctx, queryUpdate)
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				return errors.InternalServerError("error rolling back transaction")
			}
			return errors.InternalServerError("error upserting payment")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.InternalServerError("error committing transaction")
	}

	return nil
}

// FindBookingByUserID implements Repositories.
func (r *repositories) FindBookingByUserID(ctx context.Context, userID int64) (entity.Booking, error) {
	query := fmt.Sprintf(`SELECT * FROM bookings WHERE user_id = %d`, userID)
	var booking entity.Booking
	err := r.db.GetContext(ctx, &booking, query)
	if err == sql.ErrNoRows {
		return entity.Booking{}, nil
	}
	if err != nil {
		return entity.Booking{}, errors.InternalServerError("error find booking by user id")
	}
	return booking, nil
}

// FindPaymentByBookingID implements Repositories.
func (r *repositories) FindPaymentByBookingID(ctx context.Context, bookingID string) (entity.Payment, error) {
	bookingIDuuid := uuid.MustParse(bookingID)
	query := fmt.Sprintf(`SELECT * FROM payments WHERE booking_id = '%s'`, bookingIDuuid.String())
	var payment entity.Payment
	err := r.db.GetContext(ctx, &payment, query)
	if err == sql.ErrNoRows {
		return entity.Payment{}, nil
	}
	if err != nil {
		return entity.Payment{}, errors.InternalServerError("error find payment by booking id")
	}
	return payment, nil
}

func (r *repositories) ValidateToken(ctx context.Context, token string) (response.UserServiceValidate, error) {
	// http call to user service
	if !r.cb.Ready() {
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, errors.InternalServerError("error validate token")
	}
	url := fmt.Sprintf("http://%s:%s/api/private/user/validate?token=%s", r.cfg.UserService.Host, r.cfg.UserService.Port, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Invalid token: %d", resp.StatusCode))
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, errors.BadRequest("Invalid token")
	}

	// parse response
	var respBase response.BaseResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&respBase); err != nil {
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, err
	}

	respBase.Data = respBase.Data.(map[string]interface{})
	respData := response.UserServiceValidate{
		IsValid:   respBase.Data.(map[string]interface{})["is_valid"].(bool),
		UserID:    int64(respBase.Data.(map[string]interface{})["user_id"].(float64)),
		EmailUser: respBase.Data.(map[string]interface{})["email_user"].(string),
	}

	if !respData.IsValid {
		r.log.Ctx(ctx).Error("Invalid token")
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, errors.BadRequest("Invalid token")
	}

	// validate token
	return response.UserServiceValidate{
		IsValid:   respData.IsValid,
		UserID:    respData.UserID,
		EmailUser: respData.EmailUser,
	}, nil
}

type Repositories interface {
	// http
	ValidateToken(ctx context.Context, token string) (response.UserServiceValidate, error)
	SubmitPayment(ctx context.Context, bookingID string, amount float64, paymentMethod string, paymentDate time.Time) error
	InquiryTicketAmount(ctx context.Context, ticketDetailID int64, totalTicket int) (float64, error)
	DeleteTaskScheduler(ctx context.Context, taskID string) error
	// redis
	CheckStockTicket(ctx context.Context, ticketDetailID int64) (int64, error)
	DecrementStockTicket(ctx context.Context, ticketDetailID int64) error
	IncrementStockTicket(ctx context.Context, ticketDetailID int64) error
	SetTaskScheduler(ctx context.Context, expiredAt time.Duration, payload []byte) (taskID string, err error)
	// db
	UpsertBooking(ctx context.Context, booking *entity.Booking) (id string, err error)
	UpsertPayment(ctx context.Context, payment *entity.Payment) error
	FindBookingByUserID(ctx context.Context, userID int64) (entity.Booking, error)
	FindBookingByID(ctx context.Context, bookingID string) (entity.Booking, error)
	FindPaymentByBookingID(ctx context.Context, bookingID string) (entity.Payment, error)
}

func New(db *sqlx.DB, log *otelzap.Logger, httpClient *circuit.HTTPClient, redisClient *redis.Client, cfg *config.Config, clientScheduler *asynq.Client, cb *circuit.Breaker) Repositories {
	return &repositories{
		db:              db,
		log:             log,
		httpClient:      httpClient,
		redisClient:     redisClient,
		cfg:             cfg,
		clientScheduler: clientScheduler,
		cb:              cb,
	}
}

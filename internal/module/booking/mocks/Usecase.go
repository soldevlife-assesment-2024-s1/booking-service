// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import request "booking-service/internal/module/booking/models/request"
import response "booking-service/internal/module/booking/models/response"

// Usecase is an autogenerated mock type for the Usecase type
type Usecase struct {
	mock.Mock
}

// BookTicket provides a mock function with given fields: ctx, payload, userID, emailUser
func (_m *Usecase) BookTicket(ctx context.Context, payload *request.BookTicket, userID int64, emailUser string) error {
	ret := _m.Called(ctx, payload, userID, emailUser)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *request.BookTicket, int64, string) error); ok {
		r0 = rf(ctx, payload, userID, emailUser)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConsumeBookTicketQueue provides a mock function with given fields: ctx, payload
func (_m *Usecase) ConsumeBookTicketQueue(ctx context.Context, payload *request.BookTicket) error {
	ret := _m.Called(ctx, payload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *request.BookTicket) error); ok {
		r0 = rf(ctx, payload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CountPendingPayment provides a mock function with given fields: ctx, ticketID
func (_m *Usecase) CountPendingPayment(ctx context.Context, ticketID int64) (response.PendingPayment, error) {
	ret := _m.Called(ctx, ticketID)

	var r0 response.PendingPayment
	if rf, ok := ret.Get(0).(func(context.Context, int64) response.PendingPayment); ok {
		r0 = rf(ctx, ticketID)
	} else {
		r0 = ret.Get(0).(response.PendingPayment)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, ticketID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Payment provides a mock function with given fields: ctx, payload, emailUser
func (_m *Usecase) Payment(ctx context.Context, payload *request.Payment, emailUser string) error {
	ret := _m.Called(ctx, payload, emailUser)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *request.Payment, string) error); ok {
		r0 = rf(ctx, payload, emailUser)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PaymentCancel provides a mock function with given fields: ctx, payload, emailUser
func (_m *Usecase) PaymentCancel(ctx context.Context, payload *request.PaymentCancellation, emailUser string) error {
	ret := _m.Called(ctx, payload, emailUser)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *request.PaymentCancellation, string) error); ok {
		r0 = rf(ctx, payload, emailUser)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPaymentExpired provides a mock function with given fields: ctx, payload
func (_m *Usecase) SetPaymentExpired(ctx context.Context, payload *request.PaymentExpiration) error {
	ret := _m.Called(ctx, payload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *request.PaymentExpiration) error); ok {
		r0 = rf(ctx, payload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ShowBookings provides a mock function with given fields: ctx, userID
func (_m *Usecase) ShowBookings(ctx context.Context, userID int64) (response.BookedTicket, error) {
	ret := _m.Called(ctx, userID)

	var r0 response.BookedTicket
	if rf, ok := ret.Get(0).(func(context.Context, int64) response.BookedTicket); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Get(0).(response.BookedTicket)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

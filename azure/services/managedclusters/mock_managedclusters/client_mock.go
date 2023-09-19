/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by MockGen. DO NOT EDIT.
// Source: ../client.go
//
// Generated by this command:
//
//	mockgen -destination client_mock.go -package mock_managedclusters -source ../client.go CredentialGetter
//
// Package mock_managedclusters is a generated GoMock package.
package mock_managedclusters

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockCredentialGetter is a mock of CredentialGetter interface.
type MockCredentialGetter struct {
	ctrl     *gomock.Controller
	recorder *MockCredentialGetterMockRecorder
}

// MockCredentialGetterMockRecorder is the mock recorder for MockCredentialGetter.
type MockCredentialGetterMockRecorder struct {
	mock *MockCredentialGetter
}

// NewMockCredentialGetter creates a new mock instance.
func NewMockCredentialGetter(ctrl *gomock.Controller) *MockCredentialGetter {
	mock := &MockCredentialGetter{ctrl: ctrl}
	mock.recorder = &MockCredentialGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCredentialGetter) EXPECT() *MockCredentialGetterMockRecorder {
	return m.recorder
}

// GetCredentials mocks base method.
func (m *MockCredentialGetter) GetCredentials(arg0 context.Context, arg1, arg2 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCredentials", arg0, arg1, arg2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCredentials indicates an expected call of GetCredentials.
func (mr *MockCredentialGetterMockRecorder) GetCredentials(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCredentials", reflect.TypeOf((*MockCredentialGetter)(nil).GetCredentials), arg0, arg1, arg2)
}

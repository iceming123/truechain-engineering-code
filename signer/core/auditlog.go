// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"context"

	"github.com/truechain/truechain-engineering-code/common"
	"github.com/truechain/truechain-engineering-code/common/hexutil"
	"github.com/truechain/truechain-engineering-code/internal/trueapi"
	"github.com/truechain/truechain-engineering-code/log"
)

type AuditLogger struct {
	log log.Logger
	api ExternalAPI
}

func (l *AuditLogger) List(ctx context.Context) ([]common.Address, error) {
	l.log.Info("List", "type", "request", "metadata", MetadataFromContext(ctx).String())
	res, e := l.api.List(ctx)
	l.log.Info("List", "type", "response", "data", res)

	return res, e
}

func (l *AuditLogger) New(ctx context.Context) (common.Address, error) {
	return l.api.New(ctx)
}

func (l *AuditLogger) SignTransaction(ctx context.Context, args SendTxArgs, methodSelector *string) (*trueapi.SignTransactionResult, error) {
	sel := "<nil>"
	if methodSelector != nil {
		sel = *methodSelector
	}
	l.log.Info("SignTransaction", "type", "request", "metadata", MetadataFromContext(ctx).String(),
		"tx", args.String(),
		"methodSelector", sel)

	res, e := l.api.SignTransaction(ctx, args, methodSelector)
	if res != nil {
		l.log.Info("SignTransaction", "type", "response", "data", common.Bytes2Hex(res.Raw), "error", e)
	} else {
		l.log.Info("SignTransaction", "type", "response", "data", res, "error", e)
	}
	return res, e
}

func (l *AuditLogger) EcRecover(ctx context.Context, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error) {
	l.log.Info("EcRecover", "type", "request", "metadata", MetadataFromContext(ctx).String(),
		"data", common.Bytes2Hex(data), "sig", common.Bytes2Hex(sig))
	b, e := l.api.EcRecover(ctx, data, sig)
	l.log.Info("EcRecover", "type", "response", "address", b.String(), "error", e)
	return b, e
}

func (l *AuditLogger) Version(ctx context.Context) (string, error) {
	l.log.Info("Version", "type", "request", "metadata", MetadataFromContext(ctx).String())
	data, err := l.api.Version(ctx)
	l.log.Info("Version", "type", "response", "data", data, "error", err)
	return data, err

}

func NewAuditLogger(path string, api ExternalAPI) (*AuditLogger, error) {
	l := log.New("api", "signer")
	handler, err := log.FileHandler(path, log.LogfmtFormat())
	if err != nil {
		return nil, err
	}
	l.SetHandler(handler)
	l.Info("Configured", "audit log", path)
	return &AuditLogger{l, api}, nil
}

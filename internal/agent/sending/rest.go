package sending

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/compress"
	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"go.uber.org/zap"
)

type RestSender struct {
	addr string

	secretKey []byte
	encrypter encrypt.Encrypter

	// we use a custom http client here for further customization
	// and to enable connection reuse for sequential server calls
	cl      *http.Client
	cwe     compress.WriteEngine
	logger  *zap.SugaredLogger
	retrier *retrying.Executor
}

func NewRestSender(
	addr string,
	retryPolicy retrying.Policy,
	secretKey string,
	rsaKeyPath string,
	logger *zap.Logger,
) (*RestSender, error) {
	restLogger := logger.Sugar().With(zap.String("component", "rest-sender"))

	enrichedAddr, err := enrichServerAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("can't enrich address for sender to a proper format: %w", err)
	}
	restLogger.Infow("sender address for sending reports",
		"URL", enrichedAddr,
	)

	retrier := retrying.NewExecutorBuilder(retryPolicy).
		WithLogger(restLogger, "sending metrics").
		Build()

	var publicKey *rsa.PublicKey
	if rsaKeyPath != "" {
		var err error
		publicKey, err = encrypt.ReadRSAPublicKeyFromFile(rsaKeyPath)
		if err != nil {
			return nil, fmt.Errorf("can't read RSA public key for encryption: %w", err)
		}
		restLogger.Infow("using RSA public key for encryption")
	}

	rs := &RestSender{
		addr:      enrichedAddr,
		cl:        &http.Client{},
		cwe:       compress.NewGzipWriteEngine(),
		logger:    restLogger,
		retrier:   retrier,
		secretKey: []byte(secretKey),
		encrypter: encrypt.NewRSAEngineBuilder().PublicKey(publicKey).Build(),
	}
	return rs, nil
}

func (rs *RestSender) SendMetricValue(id string, mtype string, value string) error {
	return rs.retrier.Run(func() error {
		req, err := http.NewRequest(http.MethodPost, rs.composePostMetricByPathURL(id, mtype, value), nil)
		if err != nil {
			return fmt.Errorf("can't construct metric send request: %w", err)
		}
		req.Header.Add("Content-Type", "text/plain")
		encrypt.AddSignature(rs.secretKey, nil, req.Header)
		return rs.sendRequest(req)
	})
}

func (rs *RestSender) SendMetric(metric *model.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("can't construct metric update request: %w", err)
	}

	// encrypt body if needed before compression
	if rs.encrypter.EncryptingEnabled() {
		body, err = rs.encrypter.Encrypt(body)
		if err != nil {
			return fmt.Errorf("can't encrypt metric update request: %w", err)
		}
	}

	if rs.cwe != nil {
		body, err = rs.cwe.WriteFlushed(body, 0)
		if err != nil {
			return fmt.Errorf("can't write compressed metric data: %w", err)
		}
	}

	return rs.retrier.Run(func() error {
		req, err := http.NewRequest(http.MethodPost, rs.addr+"/update", bytes.NewBufferString(string(body)))
		if err != nil {
			return fmt.Errorf("can't send metric update request: %w", err)
		}
		req.Header.Add("Content-Type", "application/json")
		if rs.cwe != nil {
			rs.cwe.SetContentEncoding(req.Header)
		}
		encrypt.AddSignature(rs.secretKey, body, req.Header)
		return rs.sendRequest(req)
	})
}

func (rs *RestSender) SendMetricArray(metrics []*model.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("can't construct metrics array update request: %w", err)
	}

	// encrypt body if needed before compression
	if rs.encrypter.EncryptingEnabled() {
		body, err = rs.encrypter.Encrypt(body)
		if err != nil {
			return fmt.Errorf("can't encrypt metric update request: %w", err)
		}
	}

	if rs.cwe != nil {
		body, err = rs.cwe.WriteFlushed(body, 0)
		if err != nil {
			return fmt.Errorf("can't write compressed metrics data: %w", err)
		}
	}

	return rs.retrier.Run(func() error {
		req, err := http.NewRequest(http.MethodPost, rs.addr+"/updates", bytes.NewBufferString(string(body)))
		if err != nil {
			return fmt.Errorf("can't send metrics array update request: %w", err)
		}
		req.Header.Add("Content-Type", "application/json")
		if rs.cwe != nil {
			rs.cwe.SetContentEncoding(req.Header)
		}

		return rs.sendRequest(req)
	})
}

func (rs *RestSender) sendRequest(req *http.Request) error {
	req.Header.Set("X-Real-IP", rs.getHostIPAddr())
	resp, err := rs.cl.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to server %s: %w", rs.addr, err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			rs.logger.Error("error closing response body", zap.Error(err))
		}
	}()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return retrying.NewRetryableError(
			fmt.Errorf("can't read response body from metrics send operation: %w", err))
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error response status %d from server ", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusRequestTimeout:
		case http.StatusTooManyRequests:
		case http.StatusInternalServerError:
		case http.StatusBadGateway:
		case http.StatusServiceUnavailable:
		case http.StatusGatewayTimeout:
			return retrying.NewRetryableError(err)
		}
		return err
	}
	return nil
}

func (rs *RestSender) composePostMetricByPathURL(id string, mtype string, value string) string {
	return fmt.Sprintf("%s/update/%s/%s/%s", rs.addr, mtype, id, value)
}

func (rs *RestSender) getHostIPAddr() string {
	const loopback = "127.0.0.1"

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		rs.logger.Errorw("error getting host IP addresses", zap.Error(err))
		return loopback
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	// loopback address as a fail case
	rs.logger.Warnw("no host IP address found")
	return loopback
}

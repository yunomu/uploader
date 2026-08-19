package cfsigner

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
)

type CFSigner struct {
	domain string
	signer *sign.URLSigner
}

func New(
	domain string,
	signer *sign.URLSigner,
) *CFSigner {
	return &CFSigner{
		domain: domain,
		signer: signer,
	}
}

func (s *CFSigner) Sign(key string, expires time.Time) (string, error) {
	signedURL, err := s.signer.Sign(fmt.Sprintf("https://%s/%s", s.domain, key), expires)
	if err != nil {
		return "", err
	}

	return signedURL, nil
}

package digest

import (
	"context"
	"errors"
	"fmt"
	"github.com/containrrr/watchtower/pkg/logger"
	"github.com/containrrr/watchtower/pkg/registry/auth"
	"github.com/containrrr/watchtower/pkg/registry/manifest"
	"github.com/containrrr/watchtower/pkg/types"
	apiTypes "github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

// ContentDigestHeader is the key for the key-value pair containing the digest header
const ContentDigestHeader = "Docker-Content-Digest"

// CompareDigest ...
func CompareDigest(ctx context.Context, image apiTypes.ImageInspect, credentials *types.RegistryCredentials) (bool, error) {
	var digest string
	token, err := auth.GetToken(ctx, image, credentials)
	if err != nil {
		return false, err
	}

	digestURL, err := manifest.BuildManifestURL(image)
	if err != nil {
		return false, err
	}

	if digest, err = GetDigest(ctx, digestURL, token); err != nil {
		return false, err
	}

	logrus.WithField("remote", digest).Debug("Found a remote digest to compare with")

	if image.ID == digest {
		return true, nil
	}

	for _, dig := range image.RepoDigests {
		localDigest := strings.Split(dig, "@")[1]
		logrus.WithFields(logrus.Fields{
			"local":  localDigest,
			"remote": digest,
		}).Debug("Comparing")
		if localDigest == digest {
			logrus.Debug("Found a match")
			return true, nil
		}
	}

	return false, nil
}

// GetDigest from registry using a HEAD request to prevent rate limiting
func GetDigest(ctx context.Context, url string, token string) (string, error) {
	client := &http.Client{}
	log := logger.GetLogger(ctx).WithField("fun", "GetDigest")
	if token != "" {
		log.WithField("token", token).Debug("Setting request bearer token")
	} else {
		return "", errors.New("could not fetch token")
	}

	req, _ := http.NewRequest("HEAD", url, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "*/*")

	logrus.WithField("url", url).Debug("Doing a HEAD request to fetch a digest")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("registry responded to head request with %d", res.StatusCode)
	}
	return res.Header.Get(ContentDigestHeader), nil
}
package cosmos

import (
	"context"
	"log"
	"sync"
	"time"
)

type ctxKey string

const (
	ArtifactDir           = "/tmp/cosmos/artifacts"
	artifactoryKey ctxKey = "artifactory"
)

const (
	ArtifactSource = iota
	ArtifactDestination
	ArtifactNormalization
	ArtifactWorker
	ArtifactSrcConfig
	ArtifactDstConfig
	ArtifactCatalog
	ArtifactBeforeState
	ArtifactAfterState
	ArtifactMax
)

var ArtifactNames = [ArtifactMax]string{
	"source",
	"destination",
	"normalization",
	"worker",
	"source-config",
	"destination-config",
	"catalog",
	"before-state",
	"after-state",
}

type Artifactory struct {
	Path      string
	Once      [ArtifactMax]sync.Once
	Artifacts [ArtifactMax]*log.Logger
}

type ArtifactService interface {
	GetArtifactory(syncID int, executionDate time.Time) (*Artifactory, error)
	GetArtifactRef(artifactory *Artifactory, id int, attempt int32) (*log.Logger, error)
	WriteArtifact(artifactory *Artifactory, id int, contents interface{}) error
	GetArtifactPath(artifactory *Artifactory, id int) *string
	GetArtifactData(artifactory *Artifactory, id int) ([]byte, error)
	CloseArtifactory(artifactory *Artifactory)
}

func NewArtifactoryContext(ctx context.Context, artifactory *Artifactory) context.Context {
	return context.WithValue(ctx, artifactoryKey, artifactory)
}

func ArtifactoryFromContext(ctx context.Context) *Artifactory {
	return ctx.Value(artifactoryKey).(*Artifactory)
}

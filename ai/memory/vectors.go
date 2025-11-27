package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/upstash/vector-go"
	"go.uber.org/zap"
)

type MemoryStatus string

const (
	StatusActive     MemoryStatus = "active"
	StatusSuperseded MemoryStatus = "superseded"
	StatusDeleted    MemoryStatus = "deleted"
)

type MemoryLifespan string

const (
	LifespanShortTerm MemoryLifespan = "short_term"
	LifespanMidTerm   MemoryLifespan = "mid_term"
	LifespanLongTerm  MemoryLifespan = "long_term"
	LifespanLifelong  MemoryLifespan = "lifelong"
)

type MemoryMutability string

const (
	MutabilityMutable   MemoryMutability = "mutable"
	MutabilityImmutable MemoryMutability = "immutable"
)

type MemoryCategory string

const (
	CategoryFact       MemoryCategory = "fact"
	CategoryPreference MemoryCategory = "preference"
	CategorySkill      MemoryCategory = "skill"
	CategoryContext    MemoryCategory = "context"
	CategoryRule       MemoryCategory = "rule"
	CategoryEntity     MemoryCategory = "entity"
	CategoryEpisodic   MemoryCategory = "episodic"
)

type EntityRelationship struct {
	EntityType   string `json:"entity_type"`
	RelationType string `json:"relation_type"`
	EntityValue  string `json:"entity_value"`
}

type Memory struct {
	Id                      string               `json:"id"` // Vector Id
	SubjectKey              string               `json:"subject_key"`
	Namespace               string               `json:"namespace"`
	Category                MemoryCategory       `json:"category"`
	Summary                 string               `json:"summary"`
	RawText                 string               `json:"raw_text"`
	Importance              int                  `json:"importance"`
	Mutability              MemoryMutability     `json:"mutability"`
	Lifespan                MemoryLifespan       `json:"lifespan"`
	ForgetScore             float64              `json:"forget_score"`
	Status                  MemoryStatus         `json:"status"`
	SupersedesCanonicalKeys []string             `json:"supersedes_canonical_keys"`
	Metadata                json.RawMessage      `json:"metadata"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
	ExpiresAt               *time.Time           `json:"expires_at,omitempty"`
	EntityRelationships     []EntityRelationship `json:"entity_relationships"`
}

type VectorServices string

const (
	VectorServiceUpstash  VectorServices = "upstash"
	VectorServicePinecone VectorServices = "pinecone"
)

type filters struct {
	SearchQuery      string        `json:"search_query" description:"Query string for vector search, include common words or phrases related to the user prompt."`
	Category         *string       `json:"category,omitempty" description:"High-level category of the memory. One of: 'fact', 'preference', 'skill', 'context', 'rule', 'entity', 'episodic'.\n- fact: objective info about the subject (e.g. 'I use PostgreSQL for databases').\n- preference: likes/dislikes or choices (e.g. 'I prefer clean, readable code', 'I like Adidas').\n- skill: abilities and expertise (e.g. 'Experienced with FastAPI').\n- context: project or situation info (e.g. 'Working on e-commerce platform').\n- rule: behavioral guidelines or constraints (e.g. 'Always write tests first', 'Never reply in Telugu').\n- entity: people/organizations in subject's life (e.g. 'Jane is my mom', 'Karthik is my lead developer').\n- episodic: specific events in time (e.g. 'Yesterday we deployed the new version'). // NOTE: You can specify multiple categories by separating them with a comma (,) character. It is recommended to specify multiple categories for better results."`
	Lifespan         *string       `json:"lifespan,omitempty" description:"Intended lifespan category for this memory, typically via MemoryLifespan enum. One of:\n- 'short_term': ephemeral or near-term context (e.g. 'This week I am traveling').\n- 'mid_term': medium-lived preferences or context (e.g. 'Currently using Tailwind for styling').\n- 'long_term': persistent facts/skills (e.g. 'I use PostgreSQL', 'Experienced with FastAPI').\n- 'lifelong': identity-level traits (e.g. 'I love coding', 'I enjoy cooking').\nCombined with ForgetScore and ExpiresAt for decay and garbage collection strategies.// NOTE: You can specify multiple lifespans by separating them with a comma (,) character. It is recommended to specify multiple lifespans for better results."`
	Importance       *int          `json:"importance,omitempty" description:"Ignore this field, do not include it."`
	Expiry           *time.Time    `json:"expiry,omitempty" description:"Expiration timestamp for this memory, indicating when it should be considered stale or expired.\nUsed in conjunction with Lifespan and ForgetScore for decay and garbage collection strategies."`
	Status           *MemoryStatus `json:"status,omitempty" description:"Current lifecycle state of the memory, usually via MemoryStatus enum. Typical values:\n- 'active': Memory is current and should be considered during retrieval.\n- 'superseded': Memory has been replaced by a newer memory for the same canonical_key (e.g. old preference 'I like Adidas' after 'I don't like Adidas anymore').\n- 'deleted': Soft-deleted or logically removed memory.\nRetrieval layers generally filter to active memories by default."`
	IncludeAllScopes *bool         `json:"include_all_scopes,omitempty" description:"Ignore this field, do not include it."`
}

type v struct {
	memories Memory
	vector   []float32
}

type vectorService interface {
	upsertVectors(vectors []v) error
	queryVector(vectors []float32, topK int, fs ...filters) ([]vector.VectorScore, error)
	queryVectorByMetadata(filters filters) ([]map[string]any, error)
	updateVector(memory Memory, v ...[]float32) (bool, error)
	deleteVectors(vectorsIds []string) (count int, err error)
	shiftScope(scope string) string
	shiftUser(userId string) string
}

type vectorClient struct {
	currentService VectorServices
	client         vectorService
	logger         *zap.Logger
}

func newVectorClient(userId, scope string, logger *zap.Logger) *vectorClient {
	client := &vectorClient{
		logger: logger,
	}
	if err := client.switchService(userId, scope, VectorServiceUpstash); err != nil {
		client.logger.Error("[KARMA_MEMORY] failed to switch service", zap.Error(err))
		return nil
	}
	return client // Default service, use the useService method to set a different service
}

func (d *vectorClient) switchService(userId, scope string, service VectorServices) error {
	switch service {
	case VectorServiceUpstash:
		d.client = newUpstashClient(userId, scope, d.logger)
	case VectorServicePinecone:
		d.client = newPineconeClient(userId, scope, d.logger)
	default:
		d.logger.Error("[KARMA_MEMORY] invalid service")
		return fmt.Errorf("invalid service")
	}
	d.currentService = service
	return nil
}

func (d *vectorClient) setScope(scope string) string {
	return d.client.shiftScope(scope)
}

func (d *vectorClient) setUser(userId string) string {
	return d.client.shiftUser(userId)
}

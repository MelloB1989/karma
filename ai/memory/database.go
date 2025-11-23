package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/database"
	"github.com/MelloB1989/karma/v2/orm"
	"github.com/upstash/vector-go"
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
	ToMemoryID   string `json:"to_memory_id"`
	RelationType string `json:"relation_type"`
}

type Memory struct {
	TableName               string               `karma_table:"memories"`
	Id                      string               `json:"id" karma:"primary_key"`
	SubjectKey              string               `json:"subject_key"`
	Namespace               string               `json:"namespace"`
	Category                MemoryCategory       `json:"category"`
	Summary                 string               `json:"summary"`
	RawText                 string               `json:"raw_text"`
	CanonicalKey            *string              `json:"canonical_key,omitempty"`
	Value                   *string              `json:"value,omitempty"`
	Importance              int                  `json:"importance"`
	Mutability              MemoryMutability     `json:"mutability"`
	Lifespan                MemoryLifespan       `json:"lifespan"`
	ForgetScore             float64              `json:"forget_score"`
	Status                  MemoryStatus         `json:"status"`
	SupersedesCanonicalKeys []string             `json:"supersedes_canonical_keys" db:"supersedes_canonical_keys"`
	SupersededByID          *string              `json:"superseded_by_id,omitempty"`
	Metadata                json.RawMessage      `json:"metadata" db:"metadata"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
	ExpiresAt               *time.Time           `json:"expires_at,omitempty"`
	EntityRelationships     []EntityRelationship `json:"entity_relationships,omitempty" db:"-"` // Not stored in DB, used for link creation
}

type EntityLink struct {
	TableName    string          `karma_table:"entity_links"`
	ID           string          `json:"id" karma:"primary_key"`
	SubjectKey   string          `json:"subject_key"`
	Namespace    string          `json:"namespace"`
	FromMemoryID string          `json:"from_memory_id"`
	ToMemoryID   string          `json:"to_memory_id"`
	RelationType string          `json:"relation_type"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
}

type dbClient struct {
	userId string
	scope  string
	vn     *vector.Namespace
	vi     *vector.Index
}

func newDBClient(userId, scope string) *dbClient {
	index := vector.NewIndex(config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_URL"), config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_TOKEN"))
	userIdx := index.Namespace(userId)
	return &dbClient{
		userId: userId,
		scope:  scope,
		vn:     userIdx,
		vi:     index,
	}
}

func (d *dbClient) setScope(scope string) string {
	d.scope = scope
	return d.scope
}

func (d *dbClient) setUser(userId string) string {
	d.userId = userId
	d.vn = d.vi.Namespace(d.userId)
	return d.userId
}

func (d *dbClient) runMigrations() error {
	db, err := database.PostgresConn()
	if err != nil {
		return err
	}
	defer db.Close()

	// Create memories table with partitioning by namespace
	memoriesTable := `
	CREATE TABLE IF NOT EXISTS memories (
		id                      UUID DEFAULT gen_random_uuid(),

		-- Identify who/what this memory belongs to (no FK on purpose)
		subject_key             TEXT NOT NULL,   -- e.g. "user_123", "org_acme"

		-- Partition / namespace: app, service, or logical memory space
		namespace               TEXT NOT NULL,   -- e.g. "lyzn_chat", "sales_agent"

		-- Memory categorization (aligned with the classifier)
		category                TEXT NOT NULL CHECK (
									category IN (
										'fact',
										'preference',
										'skill',
										'context',
										'rule',
										'entity',
										'episodic'
									)
								),

		summary                 TEXT NOT NULL,
		raw_text                TEXT NOT NULL,

		-- Optional semantic key/value for facts, preferences, skills, etc.
		canonical_key           TEXT,
		value                   TEXT,

		-- Memory lifecycle & importance
		importance              INT NOT NULL CHECK (importance BETWEEN 1 AND 5),
		mutability              TEXT NOT NULL CHECK (mutability IN ('immutable', 'mutable')),
		lifespan                TEXT NOT NULL CHECK (
									lifespan IN ('short_term','mid_term','long_term','lifelong')
								),
		forget_score            REAL NOT NULL CHECK (forget_score >= 0.0 AND forget_score <= 1.0),

		-- Status & supersession (for preference changes etc.)
		status                  TEXT NOT NULL CHECK (
									status IN ('active', 'superseded', 'deleted')
								) DEFAULT 'active',
		supersedes_canonical_keys TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
		superseded_by_id        UUID,  -- references memories(id) logically; no FK constraint if you want

		metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,

		created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at              TIMESTAMPTZ,

		PRIMARY KEY (id, namespace)
	) PARTITION BY LIST (namespace);
	`

	if _, err := db.Exec(memoriesTable); err != nil {
		return err
	}

	// Create default partition for all namespaces
	defaultPartition := `
	CREATE TABLE IF NOT EXISTS memories_default
	PARTITION OF memories DEFAULT;
	`

	if _, err := db.Exec(defaultPartition); err != nil {
		return err
	}

	// Create indexes for memories table
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_memories_subject_ns_cat
		ON memories (subject_key, namespace, category, status);`,

		`CREATE INDEX IF NOT EXISTS idx_memories_expires_at
		ON memories (expires_at);`,

		`CREATE INDEX IF NOT EXISTS idx_memories_canonical_active
		ON memories (subject_key, namespace, canonical_key, status);`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	// Create entity_links table
	entityLinksTable := `
	CREATE TABLE IF NOT EXISTS entity_links (
		id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

		subject_key     TEXT NOT NULL,
		namespace       TEXT NOT NULL,

		from_memory_id  UUID NOT NULL,   -- should point to a memory with category='entity'
		to_memory_id    UUID NOT NULL,   -- another 'entity' memory
		relation_type   TEXT NOT NULL,   -- e.g. 'parent_of', 'works_with', 'cofounder_of'

		metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`

	if _, err := db.Exec(entityLinksTable); err != nil {
		return err
	}

	// Create index for entity_links table
	entityLinksIndex := `
	CREATE INDEX IF NOT EXISTS idx_entity_links_subject_ns
	ON entity_links (subject_key, namespace);
	`

	if _, err := db.Exec(entityLinksIndex); err != nil {
		return err
	}

	return nil
}

// CreateMemory inserts a new memory into the database
func (d *dbClient) createMemory(memory *Memory) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("postgres"))
	defer o.Close()

	now := time.Now()
	memory.CreatedAt = now
	memory.UpdatedAt = now
	memory.SubjectKey = d.userId
	memory.Namespace = d.scope
	if memory.Status == "" {
		memory.Status = StatusActive
	}
	if memory.Metadata == nil {
		memory.Metadata = json.RawMessage("{}")
	}
	if len(memory.SupersedesCanonicalKeys) == 0 {
		memory.SupersedesCanonicalKeys = []string{}
	}

	memory.ExpiresAt = computeExpiry(now, memory.Lifespan, memory.ForgetScore)

	if memory.Category == CategoryEntity && len(memory.EntityRelationships) > 0 {
		return o.WithTransaction(func(txOrm *orm.ORM) error {
			if err := txOrm.Insert(memory); err != nil {
				return err
			}

			return d.createEntityLinksForMemory(memory, txOrm)
		})
	}

	return o.Insert(memory)
}

// createEntityLinksForMemory creates entity links from EntityRelationships field
func (d *dbClient) createEntityLinksForMemory(memory *Memory, txOrm *orm.ORM) error {
	if memory.Category != CategoryEntity || len(memory.EntityRelationships) == 0 {
		return nil
	}

	now := time.Now()
	for _, rel := range memory.EntityRelationships {
		link := &EntityLink{
			SubjectKey:   memory.SubjectKey,
			Namespace:    memory.Namespace,
			FromMemoryID: memory.Id,
			ToMemoryID:   rel.ToMemoryID,
			RelationType: rel.RelationType,
			Metadata:     json.RawMessage("{}"),
			CreatedAt:    now,
		}

		if err := txOrm.Insert(link); err != nil {
			return err
		}
	}

	return nil
}

func computeExpiry(baseTime time.Time, lifespan MemoryLifespan, forgetScore float64) *time.Time {
	var baseDuration time.Duration

	switch lifespan {
	case LifespanShortTerm:
		baseDuration = 7 * 24 * time.Hour
	case LifespanMidTerm:
		baseDuration = 90 * 24 * time.Hour
	case LifespanLongTerm:
		baseDuration = 365 * 24 * time.Hour
	case LifespanLifelong:
		return nil
	default:
		return nil
	}

	adjustedDuration := time.Duration(float64(baseDuration) * (1.0 - forgetScore))
	expiryTime := baseTime.Add(adjustedDuration)
	return &expiryTime
}

// GetMemoryByID retrieves a memory by its ID
func (d *dbClient) getMemoryByID(id string) (*Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memory Memory
	err := o.GetByPrimaryKey(id).Scan(&memory)
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetMemoriesBySubjectAndNamespace retrieves all active memories for a subject and namespace
func (d *dbClient) getMemoriesBySubjectAndNamespace(subjectKey, namespace string) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	err := o.GetByFieldsEquals(map[string]any{
		"SubjectKey": subjectKey,
		"Namespace":  namespace,
		"Status":     StatusActive,
	}).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetMemoriesByCategory retrieves memories by category for a subject and namespace
func (d *dbClient) getMemoriesByCategory(subjectKey, namespace, category string) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	err := o.GetByFieldsEquals(map[string]any{
		"SubjectKey": subjectKey,
		"Namespace":  namespace,
		"Category":   category,
		"Status":     StatusActive,
	}).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetMemoriesByCanonicalKey retrieves memories by canonical key
func (d *dbClient) getMemoriesByCanonicalKey(subjectKey, namespace, canonicalKey string) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	err := o.GetByFieldsEquals(map[string]any{
		"SubjectKey":   subjectKey,
		"Namespace":    namespace,
		"CanonicalKey": canonicalKey,
		"Status":       StatusActive,
	}).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetMemoriesByImportance retrieves memories with importance >= the specified value
func (d *dbClient) getMemoriesByImportance(subjectKey, namespace string, minImportance int) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	query := `SELECT * FROM memories WHERE subject_key = $1 AND namespace = $2 AND importance >= $3 AND status = $4`
	err := o.QueryRaw(query, subjectKey, namespace, minImportance, StatusActive).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetMemoriesByLifespan retrieves memories by lifespan
func (d *dbClient) getMemoriesByLifespan(subjectKey, namespace, lifespan string) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	err := o.GetByFieldsEquals(map[string]any{
		"SubjectKey": subjectKey,
		"Namespace":  namespace,
		"Lifespan":   lifespan,
		"Status":     StatusActive,
	}).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetExpiredMemories retrieves memories that have expired
func (d *dbClient) getExpiredMemories() ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	query := `SELECT * FROM memories WHERE expires_at IS NOT NULL AND expires_at < NOW() AND status = $1`
	err := o.QueryRaw(query, StatusActive).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// SearchMemoriesByText searches memories by text content (summary or raw_text)
func (d *dbClient) searchMemoriesByText(subjectKey, namespace, searchText string) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	pattern := fmt.Sprintf("%%%s%%", searchText)
	query := `SELECT * FROM memories WHERE subject_key = $1 AND namespace = $2 AND status = $3 AND (summary ILIKE $4 OR raw_text ILIKE $4)`
	err := o.QueryRaw(query, subjectKey, namespace, StatusActive, pattern).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// UpdateMemory updates an existing memory
func (d *dbClient) updateMemory(memory *Memory) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	memory.UpdatedAt = time.Now()
	return o.Update(memory, memory.Id)
}

// UpdateMemoryStatus updates the status of a memory
func (d *dbClient) updateMemoryStatus(id, status string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	query := `UPDATE memories SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := o.ExecuteRaw(query, status, id)
	return err
}

// SupersedeMemory marks old memories as superseded and creates a new one
func (d *dbClient) supersedeMemory(newMemory *Memory, oldCanonicalKeys []string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	return o.WithTransaction(func(txOrm *orm.ORM) error {
		// Mark old memories as superseded
		for _, oldKey := range oldCanonicalKeys {
			query := `UPDATE memories SET status = $1, superseded_by_id = $2, updated_at = NOW()
					  WHERE subject_key = $3 AND namespace = $4 AND canonical_key = $5 AND status = $6`
			_, err := txOrm.ExecuteRaw(query, StatusSuperseded, newMemory.Id, newMemory.SubjectKey,
				newMemory.Namespace, oldKey, StatusActive)
			if err != nil {
				return err
			}
		}

		// Insert new memory
		newMemory.SupersedesCanonicalKeys = oldCanonicalKeys
		now := time.Now()
		newMemory.CreatedAt = now
		newMemory.UpdatedAt = now
		return txOrm.Insert(newMemory)
	})
}

// DeleteMemory soft deletes a memory by marking it as deleted
func (d *dbClient) deleteMemory(id string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	query := `UPDATE memories SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := o.ExecuteRaw(query, StatusDeleted, id)
	return err
}

// DeleteMemoriesByCanonicalKey soft deletes all memories with a specific canonical key
func (d *dbClient) deleteMemoriesByCanonicalKey(subjectKey, namespace, canonicalKey string) (int64, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	query := `UPDATE memories SET status = $1, updated_at = NOW()
			  WHERE subject_key = $2 AND namespace = $3 AND canonical_key = $4 AND status = $5`
	result, err := o.ExecuteRaw(query, StatusDeleted, subjectKey, namespace, canonicalKey, StatusActive)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// HardDeleteMemory permanently removes a memory from the database
func (d *dbClient) hardDeleteMemory(id string) (int64, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	return o.DeleteByPrimaryKey(id)
}

// IncrementForgetScore increments the forget score for a memory
func (d *dbClient) incrementForgetScore(id string, increment float64) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	query := `UPDATE memories SET forget_score = LEAST(forget_score + $1, 1.0), updated_at = NOW() WHERE id = $2`
	_, err := o.ExecuteRaw(query, increment, id)
	return err
}

// ResetForgetScore resets the forget score for a memory (when accessed)
func (d *dbClient) resetForgetScore(id string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	query := `UPDATE memories SET forget_score = 0.0, updated_at = NOW() WHERE id = $1`
	_, err := o.ExecuteRaw(query, id)
	return err
}

// GetMemoriesByForgetScore retrieves memories with forget_score >= threshold
func (d *dbClient) getMemoriesByForgetScore(subjectKey, namespace string, threshold float64) ([]Memory, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var memories []Memory
	query := `SELECT * FROM memories WHERE subject_key = $1 AND namespace = $2 AND forget_score >= $3 AND status = $4`
	err := o.QueryRaw(query, subjectKey, namespace, threshold, StatusActive).Scan(&memories)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// CreateEntityLink creates a link between two entity memories
// This validates that both memories exist and are of category 'entity'
func (d *dbClient) createEntityLink(link *EntityLink) error {
	o := orm.Load(&EntityLink{}, orm.WithDatabasePrefix("postgres"))
	defer o.Close()

	// Validate that both memories exist and are entity type
	if err := d.validateEntityMemories(link.FromMemoryID, link.ToMemoryID); err != nil {
		return err
	}

	link.CreatedAt = time.Now()
	if link.Metadata == nil {
		link.Metadata = json.RawMessage("{}")
	}

	return o.Insert(link)
}

// validateEntityMemories checks that both memory IDs exist and are of category 'entity'
func (d *dbClient) validateEntityMemories(fromMemoryID, toMemoryID string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("postgres"))
	defer o.Close()

	// Check from_memory
	var fromMemory Memory
	if err := o.GetByPrimaryKey(fromMemoryID).Scan(&fromMemory); err != nil {
		return fmt.Errorf("from_memory_id %s not found: %w", fromMemoryID, err)
	}
	if fromMemory.Category != CategoryEntity {
		return fmt.Errorf("from_memory_id %s is not an entity (category: %s)", fromMemoryID, fromMemory.Category)
	}

	// Check to_memory
	var toMemory Memory
	if err := o.GetByPrimaryKey(toMemoryID).Scan(&toMemory); err != nil {
		return fmt.Errorf("to_memory_id %s not found: %w", toMemoryID, err)
	}
	if toMemory.Category != CategoryEntity {
		return fmt.Errorf("to_memory_id %s is not an entity (category: %s)", toMemoryID, toMemory.Category)
	}

	return nil
}

// GetEntityLinks retrieves all links for an entity memory
func (d *dbClient) getEntityLinks(subjectKey, namespace, memoryID string) ([]EntityLink, error) {
	o := orm.Load(&EntityLink{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var links []EntityLink
	query := `SELECT * FROM entity_links WHERE subject_key = $1 AND namespace = $2 AND (from_memory_id = $3 OR to_memory_id = $3)`
	err := o.QueryRaw(query, subjectKey, namespace, memoryID).Scan(&links)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// GetEntityLinksByRelationType retrieves links filtered by relation type
func (d *dbClient) getEntityLinksByRelationType(subjectKey, namespace, memoryID, relationType string) ([]EntityLink, error) {
	o := orm.Load(&EntityLink{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	var links []EntityLink
	query := `SELECT * FROM entity_links WHERE subject_key = $1 AND namespace = $2 AND (from_memory_id = $3 OR to_memory_id = $3) AND relation_type = $4`
	err := o.QueryRaw(query, subjectKey, namespace, memoryID, relationType).Scan(&links)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// DeleteEntityLink deletes a specific entity link
func (d *dbClient) deleteEntityLink(id string) (int64, error) {
	o := orm.Load(&EntityLink{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	return o.DeleteByPrimaryKey(id)
}

// DeleteEntityLinksByMemoryID deletes all links associated with a memory
// This should be called when deleting an entity memory
func (d *dbClient) deleteEntityLinksByMemoryID(memoryID string) (int64, error) {
	o := orm.Load(&EntityLink{}, orm.WithDatabasePrefix("postgres"))
	defer o.Close()

	query := `DELETE FROM entity_links WHERE from_memory_id = $1 OR to_memory_id = $1`
	result, err := o.ExecuteRaw(query, memoryID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteMemoryWithLinks deletes a memory and its associated entity links (for entity category)
func (d *dbClient) deleteMemoryWithLinks(id string) error {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("postgres"))
	defer o.Close()

	return o.WithTransaction(func(txOrm *orm.ORM) error {
		// Get the memory to check if it's an entity
		var memory Memory
		if err := txOrm.GetByPrimaryKey(id).Scan(&memory); err != nil {
			return err
		}

		// If it's an entity, delete associated links first
		if memory.Category == CategoryEntity {
			query := `DELETE FROM entity_links WHERE from_memory_id = $1 OR to_memory_id = $1`
			if _, err := txOrm.ExecuteRaw(query, id); err != nil {
				return err
			}
		}

		// Soft delete the memory
		query := `UPDATE memories SET status = $1, updated_at = NOW() WHERE id = $2`
		_, err := txOrm.ExecuteRaw(query, StatusDeleted, id)
		return err
	})
}

// GetMemoryCount returns the count of memories for a subject and namespace
func (d *dbClient) getMemoryCount(subjectKey, namespace string) (int, error) {
	o := orm.Load(&Memory{}, orm.WithDatabasePrefix("KARMA_MEMORY"))
	defer o.Close()

	return o.GetCount(map[string]any{
		"SubjectKey": subjectKey,
		"Namespace":  namespace,
		"Status":     StatusActive,
	})
}

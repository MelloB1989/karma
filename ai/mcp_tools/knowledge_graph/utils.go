package knowledgegraph

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/MelloB1989/karma/config"
)

var MEMORY_PATH = config.GetEnvRaw("KG_MCP_MEMORY_PATH")

type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entity_type"`
	Observations []string `json:"observations"`
}

type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relation_type"`
}

type KnowledgeGraph struct {
	// Core data storage
	entities  map[string]*Entity   `json:"-"`
	relations map[string]*Relation `json:"-"`

	// Indexes for efficient querying
	entitiesByType  map[string][]string `json:"-"`
	relationsByType map[string][]string `json:"-"`
	relationsByFrom map[string][]string `json:"-"`
	relationsByTo   map[string][]string `json:"-"`

	// For JSON serialization compatibility
	EntitiesSlice  []Entity   `json:"entities"`
	RelationsSlice []Relation `json:"relations"`

	// Thread safety
	mutex sync.RWMutex `json:"-"`
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		entities:        make(map[string]*Entity),
		relations:       make(map[string]*Relation),
		entitiesByType:  make(map[string][]string),
		relationsByType: make(map[string][]string),
		relationsByFrom: make(map[string][]string),
		relationsByTo:   make(map[string][]string),
	}
}

func (kg *KnowledgeGraph) LoadGraph() (*KnowledgeGraph, error) {
	kg.mutex.Lock()
	defer kg.mutex.Unlock()

	file, err := os.Open(MEMORY_PATH)
	if err != nil {
		log.Printf("Error opening knowledge graph memory file: %v", err)
		return kg, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var tempGraph struct {
		Entities  []Entity   `json:"entities"`
		Relations []Relation `json:"relations"`
	}

	if err := decoder.Decode(&tempGraph); err != nil {
		log.Printf("Error decoding knowledge graph memory file: %v", err)
		return kg, err
	}

	// Initialize the improved data structures
	kg.entities = make(map[string]*Entity)
	kg.relations = make(map[string]*Relation)
	kg.entitiesByType = make(map[string][]string)
	kg.relationsByType = make(map[string][]string)
	kg.relationsByFrom = make(map[string][]string)
	kg.relationsByTo = make(map[string][]string)

	// Populate entities
	for _, entity := range tempGraph.Entities {
		entityCopy := entity
		kg.entities[entity.Name] = &entityCopy
		kg.entitiesByType[entity.EntityType] = append(kg.entitiesByType[entity.EntityType], entity.Name)
	}

	// Populate relations
	for _, relation := range tempGraph.Relations {
		relationCopy := relation
		key := kg.relationKey(relation.From, relation.To, relation.RelationType)
		kg.relations[key] = &relationCopy

		kg.relationsByType[relation.RelationType] = append(kg.relationsByType[relation.RelationType], key)
		kg.relationsByFrom[relation.From] = append(kg.relationsByFrom[relation.From], key)
		kg.relationsByTo[relation.To] = append(kg.relationsByTo[relation.To], key)
	}

	return kg, nil
}

// SaveGraph saves the knowledge graph to file
func (kg *KnowledgeGraph) SaveGraph() error {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	file, err := os.Create(MEMORY_PATH)
	if err != nil {
		log.Printf("Error creating knowledge graph memory file: %v", err)
		return err
	}
	defer file.Close()

	// Convert maps back to slices for JSON serialization
	kg.EntitiesSlice = make([]Entity, 0, len(kg.entities))
	for _, entity := range kg.entities {
		kg.EntitiesSlice = append(kg.EntitiesSlice, *entity)
	}

	kg.RelationsSlice = make([]Relation, 0, len(kg.relations))
	for _, relation := range kg.relations {
		kg.RelationsSlice = append(kg.RelationsSlice, *relation)
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(kg); err != nil {
		log.Printf("Error encoding knowledge graph memory file: %v", err)
		return err
	}

	return nil
}

func (kg *KnowledgeGraph) CreateEntity(entities []Entity) *KnowledgeGraph {
	kg.mutex.Lock()
	defer kg.mutex.Unlock()

	for _, entity := range entities {
		entityCopy := entity
		kg.entities[entity.Name] = &entityCopy
		kg.entitiesByType[entity.EntityType] = append(kg.entitiesByType[entity.EntityType], entity.Name)
	}

	return kg
}

func (kg *KnowledgeGraph) CreateRelations(relations []Relation) *KnowledgeGraph {
	kg.mutex.Lock()
	defer kg.mutex.Unlock()

	for _, relation := range relations {
		relationCopy := relation
		key := kg.relationKey(relation.From, relation.To, relation.RelationType)
		kg.relations[key] = &relationCopy

		kg.relationsByType[relation.RelationType] = append(kg.relationsByType[relation.RelationType], key)
		kg.relationsByFrom[relation.From] = append(kg.relationsByFrom[relation.From], key)
		kg.relationsByTo[relation.To] = append(kg.relationsByTo[relation.To], key)
	}

	return kg
}

func (kg *KnowledgeGraph) AddObservations(entityName string, contents []string) *KnowledgeGraph {
	kg.mutex.Lock()
	defer kg.mutex.Unlock()

	if entity, exists := kg.entities[entityName]; exists {
		entity.Observations = append(entity.Observations, contents...)
	}

	return kg
}

func (kg *KnowledgeGraph) DeleteEntities(entityNames []string) *KnowledgeGraph {
	kg.mutex.Lock()
	defer kg.mutex.Unlock()

	for _, entityName := range entityNames {
		if entity, exists := kg.entities[entityName]; exists {
			// Remove from entities map
			delete(kg.entities, entityName)

			// Remove from entity type index
			kg.removeFromSlice(kg.entitiesByType[entity.EntityType], entityName)

			// Remove all relations involving this entity
			kg.deleteEntityRelations(entityName)
		}
	}

	return kg
}

func (kg *KnowledgeGraph) GetEntity(name string) (*Entity, bool) {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	entity, exists := kg.entities[name]
	if !exists {
		return nil, false
	}

	entityCopy := *entity
	return &entityCopy, true
}

func (kg *KnowledgeGraph) GetEntitiesByType(entityType string) []Entity {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	entityNames := kg.entitiesByType[entityType]
	result := make([]Entity, 0, len(entityNames))

	for _, name := range entityNames {
		if entity, exists := kg.entities[name]; exists {
			result = append(result, *entity)
		}
	}

	return result
}

func (kg *KnowledgeGraph) GetRelationsByEntity(entityName string) []Relation {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	var result []Relation

	for _, key := range kg.relationsByFrom[entityName] {
		if relation, exists := kg.relations[key]; exists {
			result = append(result, *relation)
		}
	}

	// Get relations where entity is the target
	for _, key := range kg.relationsByTo[entityName] {
		if relation, exists := kg.relations[key]; exists {
			// Avoid duplicates for self-relations
			isDuplicate := false
			for _, existing := range result {
				if existing.From == relation.From && existing.To == relation.To && existing.RelationType == relation.RelationType {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				result = append(result, *relation)
			}
		}
	}

	return result
}

func (kg *KnowledgeGraph) GetRelationsByType(relationType string) []Relation {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	relationKeys := kg.relationsByType[relationType]
	result := make([]Relation, 0, len(relationKeys))

	for _, key := range relationKeys {
		if relation, exists := kg.relations[key]; exists {
			result = append(result, *relation)
		}
	}

	return result
}

func (kg *KnowledgeGraph) EntityExists(name string) bool {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	_, exists := kg.entities[name]
	return exists
}

func (kg *KnowledgeGraph) RelationExists(from, to, relationType string) bool {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	key := kg.relationKey(from, to, relationType)
	_, exists := kg.relations[key]
	return exists
}

func (kg *KnowledgeGraph) GetStats() map[string]any {
	kg.mutex.RLock()
	defer kg.mutex.RUnlock()

	entityTypeCount := make(map[string]int)
	for entityType, entities := range kg.entitiesByType {
		entityTypeCount[entityType] = len(entities)
	}

	relationTypeCount := make(map[string]int)
	for relationType, relations := range kg.relationsByType {
		relationTypeCount[relationType] = len(relations)
	}

	return map[string]any{
		"total_entities":  len(kg.entities),
		"total_relations": len(kg.relations),
		"entity_types":    entityTypeCount,
		"relation_types":  relationTypeCount,
	}
}

// Helper methods

func (kg *KnowledgeGraph) relationKey(from, to, relationType string) string {
	return fmt.Sprintf("%s->%s:%s", from, to, relationType)
}

func (kg *KnowledgeGraph) deleteEntityRelations(entityName string) {
	// Get all relation keys involving this entity
	var keysToDelete []string

	for _, key := range kg.relationsByFrom[entityName] {
		keysToDelete = append(keysToDelete, key)
	}

	for _, key := range kg.relationsByTo[entityName] {
		keysToDelete = append(keysToDelete, key)
	}

	// Remove duplicates and delete relations
	seen := make(map[string]bool)
	for _, key := range keysToDelete {
		if !seen[key] {
			seen[key] = true
			if relation, exists := kg.relations[key]; exists {
				kg.removeRelationFromIndexes(key, relation)
				delete(kg.relations, key)
			}
		}
	}

	// Clean up the entity from relation indexes
	delete(kg.relationsByFrom, entityName)
	delete(kg.relationsByTo, entityName)
}

func (kg *KnowledgeGraph) removeRelationFromIndexes(key string, relation *Relation) {
	// Remove from type index
	kg.removeFromSlice(kg.relationsByType[relation.RelationType], key)

	// Remove from from/to indexes
	kg.removeFromSlice(kg.relationsByFrom[relation.From], key)
	kg.removeFromSlice(kg.relationsByTo[relation.To], key)
}

func (kg *KnowledgeGraph) removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	knowledgegraph "server/kg"
	"time"

	"github.com/MelloB1989/karma/ai/mcp"
	mc "github.com/mark3labs/mcp-go/mcp"
)

var graphs map[string]*knowledgegraph.KnowledgeGraph

func init() {
	graphs = make(map[string]*knowledgegraph.KnowledgeGraph)
}

func main() {
	kgMCP := mcp.NewMCPServer("Knowledge Graph", "1.0.0",
		mcp.WithDebug(true),
		mcp.WithRateLimit(mcp.RateLimit{Limit: 10, Window: time.Minute * 1}),
		mcp.WithAuthentication(false),
		mcp.WithLogging(true),
		mcp.WithPort(8080),
		mcp.WithEndpoint("mcp"),
		mcp.WithTools(
			loadKgTool(),
			createEntityTool(),
			createRelationsTool(),
			addObservationsTool(),
			deleteEntitesTool(),
			getEntityTool(),
			getEntitiesByTypeTool(),
			getRelationsByEntityTool(),
			getRelationsByTypeTool(),
			entityExistsTool(),
			relationExistsTool(),
			getStatsTool(),
			saveGraphTool(),
		),
	)

	kgMCP.Start()
}

func loadKgTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"load_kg",
			mc.WithDescription("Load or create a knowledge graph. Returns the graph status."),
			mc.WithString("graph_id",
				mc.Description("Optional unique ID of the graph. If not provided, creates a new graph with generated ID."),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")

			var kg *knowledgegraph.KnowledgeGraph

			if gid == "" {
				// Create new graph with generated ID
				kg = knowledgegraph.NewKnowledgeGraph()
				gid = kg.Name
			} else if existingKg, exists := graphs[gid]; exists {
				kg = existingKg
			} else {
				kg = knowledgegraph.NewKnowledgeGraph()
			}

			graphs[gid] = kg
			stats := kg.GetStats()

			result := map[string]any{
				"graph_id": gid,
				"status":   "loaded",
				"stats":    stats,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func createEntityTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"create_entity",
			mc.WithDescription("Create one or more entities in the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to modify"),
				mc.Required(),
			),
			mc.WithString("entities",
				mc.Description("JSON array of entities to create. Each entity should have 'name', 'entity_type', and optional 'observations' array"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entitiesJson := request.GetString("entities", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			var entities []knowledgegraph.Entity
			if err := json.Unmarshal([]byte(entitiesJson), &entities); err != nil {
				return mc.NewToolResultError(fmt.Sprintf("Invalid entities JSON: %v", err)), nil
			}

			kg.CreateEntity(entities)

			result := map[string]any{
				"status":        "success",
				"created_count": len(entities),
				"message":       fmt.Sprintf("Successfully created %d entities", len(entities)),
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func createRelationsTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"create_relations",
			mc.WithDescription("Create one or more relations in the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to modify"),
				mc.Required(),
			),
			mc.WithString("relations",
				mc.Description("JSON array of relations to create. Each relation should have 'from', 'to', and 'relation_type'"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			relationsJson := request.GetString("relations", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			var relations []knowledgegraph.Relation
			if err := json.Unmarshal([]byte(relationsJson), &relations); err != nil {
				return mc.NewToolResultError(fmt.Sprintf("Invalid relations JSON: %v", err)), nil
			}

			kg.CreateRelations(relations)

			result := map[string]any{
				"status":        "success",
				"created_count": len(relations),
				"message":       fmt.Sprintf("Successfully created %d relations", len(relations)),
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func addObservationsTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"add_observations",
			mc.WithDescription("Add observations to an existing entity"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to modify"),
				mc.Required(),
			),
			mc.WithString("entity_name",
				mc.Description("Name of the entity to add observations to"),
				mc.Required(),
			),
			mc.WithString("observations",
				mc.Description("JSON array of observation strings to add"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityName := request.GetString("entity_name", "")
			observationsJson := request.GetString("observations", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			var observations []string
			if err := json.Unmarshal([]byte(observationsJson), &observations); err != nil {
				return mc.NewToolResultError(fmt.Sprintf("Invalid observations JSON: %v", err)), nil
			}

			if !kg.EntityExists(entityName) {
				return mc.NewToolResultError(fmt.Sprintf("Entity '%s' not found", entityName)), nil
			}

			kg.AddObservations(entityName, observations)

			result := map[string]any{
				"status":      "success",
				"entity_name": entityName,
				"added_count": len(observations),
				"message":     fmt.Sprintf("Successfully added %d observations to %s", len(observations), entityName),
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func deleteEntitesTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"delete_entities",
			mc.WithDescription("Delete entities and their associated relations from the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to modify"),
				mc.Required(),
			),
			mc.WithString("entity_names",
				mc.Description("JSON array of entity names to delete"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityNamesJson := request.GetString("entity_names", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			var entityNames []string
			if err := json.Unmarshal([]byte(entityNamesJson), &entityNames); err != nil {
				return mc.NewToolResultError(fmt.Sprintf("Invalid entity_names JSON: %v", err)), nil
			}

			kg.DeleteEntities(entityNames)

			result := map[string]any{
				"status":        "success",
				"deleted_count": len(entityNames),
				"message":       fmt.Sprintf("Successfully deleted %d entities", len(entityNames)),
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func getEntityTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"get_entity",
			mc.WithDescription("Retrieve a specific entity by name"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("entity_name",
				mc.Description("Name of the entity to retrieve"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityName := request.GetString("entity_name", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			entity, found := kg.GetEntity(entityName)
			if !found {
				return mc.NewToolResultError(fmt.Sprintf("Entity '%s' not found", entityName)), nil
			}

			jsonResult, _ := json.Marshal(entity)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func getEntitiesByTypeTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"get_entities_by_type",
			mc.WithDescription("Retrieve all entities of a specific type"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("entity_type",
				mc.Description("Type of entities to retrieve"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityType := request.GetString("entity_type", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			entities := kg.GetEntitiesByType(entityType)

			result := map[string]any{
				"entity_type": entityType,
				"count":       len(entities),
				"entities":    entities,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func getRelationsByEntityTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"get_relations_by_entity",
			mc.WithDescription("Retrieve all relations involving a specific entity"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("entity_name",
				mc.Description("Name of the entity to find relations for"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityName := request.GetString("entity_name", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			relations := kg.GetRelationsByEntity(entityName)

			result := map[string]any{
				"entity_name": entityName,
				"count":       len(relations),
				"relations":   relations,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func getRelationsByTypeTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"get_relations_by_type",
			mc.WithDescription("Retrieve all relations of a specific type"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("relation_type",
				mc.Description("Type of relations to retrieve"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			relationType := request.GetString("relation_type", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			relations := kg.GetRelationsByType(relationType)

			result := map[string]any{
				"relation_type": relationType,
				"count":         len(relations),
				"relations":     relations,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func entityExistsTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"entity_exists",
			mc.WithDescription("Check if an entity exists in the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("entity_name",
				mc.Description("Name of the entity to check"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			entityName := request.GetString("entity_name", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			exists = kg.EntityExists(entityName)

			result := map[string]any{
				"entity_name": entityName,
				"exists":      exists,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func relationExistsTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"relation_exists",
			mc.WithDescription("Check if a specific relation exists in the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
			mc.WithString("from_entity",
				mc.Description("Source entity of the relation"),
				mc.Required(),
			),
			mc.WithString("to_entity",
				mc.Description("Target entity of the relation"),
				mc.Required(),
			),
			mc.WithString("relation_type",
				mc.Description("Type of the relation"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")
			fromEntity := request.GetString("from_entity", "")
			toEntity := request.GetString("to_entity", "")
			relationType := request.GetString("relation_type", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			exists = kg.RelationExists(fromEntity, toEntity, relationType)

			result := map[string]any{
				"from_entity":   fromEntity,
				"to_entity":     toEntity,
				"relation_type": relationType,
				"exists":        exists,
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func getStatsTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"get_stats",
			mc.WithDescription("Get statistics about the knowledge graph"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to query"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			stats := kg.GetStats()

			jsonResult, _ := json.Marshal(stats)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

func saveGraphTool() mcp.Tool {
	return mcp.Tool{
		Tool: mc.NewTool(
			"save_graph",
			mc.WithDescription("Save the knowledge graph to persistent storage"),
			mc.WithString("graph_id",
				mc.Description("ID of the graph to save"),
				mc.Required(),
			),
		),
		Handler: func(ctx context.Context, request mc.CallToolRequest) (*mc.CallToolResult, error) {
			gid := request.GetString("graph_id", "")

			if gid == "" {
				return mc.NewToolResultError("graph_id is required"), nil
			}

			kg, exists := graphs[gid]
			if !exists {
				return mc.NewToolResultError("Graph not found. Use load_kg first."), nil
			}

			if err := kg.SaveGraph(); err != nil {
				return mc.NewToolResultError(fmt.Sprintf("Failed to save graph: %v", err)), nil
			}

			result := map[string]any{
				"status":   "success",
				"graph_id": gid,
				"message":  "Graph saved successfully",
			}

			jsonResult, _ := json.Marshal(result)
			return mc.NewToolResultText(string(jsonResult)), nil
		},
	}
}

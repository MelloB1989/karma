package payments

import (
	"context"
	"encoding/json"
	"log"

	"github.com/MelloB1989/karma/config"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func PushOrderToRedis(Order RedisOrder) {
	opt, _ := redis.ParseURL(config.DefaultConfig().RedisURL)
	client := redis.NewClient(opt)

	// Stringify the RedisOrder struct
	orderJSON, err := json.Marshal(Order)
	if err != nil {
		// log.Fatalln("Error stringifying order:", err)
		log.Println(err)
	}

	client.Set(ctx, Order.OrderID, orderJSON, 0)
}

func GetOrderFromRedis(OrderID string) (RedisOrder, error) {
	opt, err := redis.ParseURL(config.DefaultConfig().RedisURL)
	if err != nil {
		// log.Fatalln("Error parsing Redis URL:", err)
		log.Println(err)
		return RedisOrder{}, err
	}
	client := redis.NewClient(opt)

	ctx := context.Background()
	orderJSON, err := client.Get(ctx, OrderID).Result()
	if err != nil {
		log.Println("Error getting order from Redis:", err)
		return RedisOrder{}, err
	}

	var order RedisOrder
	err = json.Unmarshal([]byte(orderJSON), &order)
	if err != nil {
		log.Println("Error unmarshalling order:", err)
		return RedisOrder{}, err
	}

	return order, nil
}

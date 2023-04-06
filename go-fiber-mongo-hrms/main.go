package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance
var dbName = "fiber-hrms"

func main() {

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoUri))

	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	db := client.Database(dbName)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Disconnect(ctx)

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(databases)

	app := fiber.New()

	app.Get("/employee", GetEmployee)
	app.Post("/employee", AddEmployee)
	app.Put("/employee/:id", EditEmployee)
	app.Delete("/employee/:id", DeleteEmployee)

	log.Fatal(app.Listen(":3000"))

}

const mongoUri = "mongodb+srv://<Username>:<Password>@clustergo.vheaban.mongodb.net/?retryWrites=true&w=majority"

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	NAME   string  `json:"name"`
	SALARY float64 `json:"salary"`
	AGE    float64 `json:"age"`
}

func GetEmployee(c *fiber.Ctx) {

	query := bson.D{{}}

	cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)

	if err != nil {
		c.Status(500).SendString(err.Error())
	}

	var employees []Employee = make([]Employee, 0)

	if err := cursor.All(c.Context(), &employees); err != nil {
		c.Status(500).SendString(err.Error())
	}

	c.JSON(employees)

}

func AddEmployee(c *fiber.Ctx) {

	collection := mg.Db.Collection("employees")
	employee := new(Employee)

	if err := c.BodyParser(employee); err != nil {
		c.Status(400).SendString(err.Error())
	}

	employee.ID = ""

	insertionResult, err := collection.InsertOne(c.Context(), employee)

	if err != nil {
		c.Status(500).SendString(err.Error())
	}

	filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
	createdRecord := collection.FindOne(c.Context(), filter)

	createdEmployee := &Employee{}
	createdRecord.Decode(createdEmployee)
	c.Status(201).JSON(createdEmployee)

}

func EditEmployee(c *fiber.Ctx) {

	id := c.Params("id")

	employeeId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.Status(400).SendString(err.Error())
	}

	var employee = Employee{}

	if err := c.BodyParser(employee); err != nil {
		c.Status(400).SendString(err.Error())
	}

	query := bson.D{{Key: "_id", Value: employeeId}}
	update := bson.D{{
		Key: "$set",
		Value: bson.D{
			{Key: "name", Value: employee.NAME},
			{Key: "age", Value: employee.AGE},
			{Key: "salary", Value: employee.SALARY},
		},
	}}

	err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.Status(400).SendString(err.Error())
		}

	}

	employee.ID = id
	c.Status(201).JSON(employee)

}

func DeleteEmployee(c *fiber.Ctx) {

	id := c.Params("id")
	employeeId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.Status(400).SendString(err.Error())
	}

	query := bson.D{{Key: "_id", Value: employeeId}}

	result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

	if err != nil {
		c.Status(400)
	}

	if result.DeletedCount < 1 {
		c.Status(404)
	}
	c.Status(200).JSON("Record Deleted")
}

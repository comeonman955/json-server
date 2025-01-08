package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

// Product struct
type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Brand       string             `bson:"brand" json:"brand"`
	Price       float64            `bson:"price" json:"price"`
	Description string             `bson:"description" json:"description"`
	Stock       int                `bson:"stock" json:"stock"`
}

// Connect to MongoDB
func connectToMongoDB() {
	uri := "mongodb://127.0.0.1:27017"

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Error creating MongoDB client:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Could not connect to MongoDB:", err)
	}

	db = client.Database("supplement_shop")
	fmt.Println("Connected to MongoDB successfully")
}

// Enable CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Create a new product
func createProductHandler(w http.ResponseWriter, r *http.Request) {
	var product Product
	err := json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collectionName := r.URL.Query().Get("collection")
	if collectionName == "" {
		http.Error(w, "Collection name is required", http.StatusBadRequest)
		return
	}

	collection := db.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	product.ID = primitive.NewObjectID() // Automatically generate a new ID for the product
	_, err = collection.InsertOne(ctx, product)
	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusCreated, product)
}

// Get a product by ID
func getProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	collectionName := r.URL.Query().Get("collection")
	if collectionName == "" {
		http.Error(w, "Collection name is required", http.StatusBadRequest)
		return
	}

	collection := db.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var product Product
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	respondWithJSON(w, http.StatusOK, product)
}

// Update a product by ID
func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var updatedProduct Product
	err = json.NewDecoder(r.Body).Decode(&updatedProduct)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collectionName := r.URL.Query().Get("collection")
	if collectionName == "" {
		http.Error(w, "Collection name is required", http.StatusBadRequest)
		return
	}

	collection := db.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updatedProduct})
	if err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedProduct)
}

// Delete a product by ID
func deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	collectionName := r.URL.Query().Get("collection")
	if collectionName == "" {
		http.Error(w, "Collection name is required", http.StatusBadRequest)
		return
	}

	collection := db.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		http.Error(w, "Failed to delete product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get all products from a collection
func getAllProductsHandler(w http.ResponseWriter, r *http.Request) {
	collectionName := r.URL.Query().Get("collection")
	if collectionName == "" {
		http.Error(w, "Collection name is required", http.StatusBadRequest)
		return
	}

	collection := db.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var products []Product
	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			http.Error(w, "Failed to decode product", http.StatusInternalServerError)
			return
		}
		products = append(products, product)
	}

	if err := cursor.Err(); err != nil {
		http.Error(w, "Cursor error", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, products)
}

// Respond with JSON
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// Main function
func main() {
	connectToMongoDB()

	// Create a new ServeMux and attach the handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/create", createProductHandler)
	mux.HandleFunc("/get", getProductHandler)
	mux.HandleFunc("/update", updateProductHandler)
	mux.HandleFunc("/delete", deleteProductHandler)
	mux.HandleFunc("/products", getAllProductsHandler) // New route for fetching products

	// Apply CORS middleware to all routes
	log.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", enableCORS(mux))
}

package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"golangify.com/snippetbox/products"
)

var (
	marketTpl       = template.Must(template.ParseFiles("index/market.html"))
	aboutTpl        = template.Must(template.ParseFiles("index/about.html"))
	clothingTpl     = template.Must(template.ParseFiles("index/product.html"))
	accessoryTpl    = template.Must(template.ParseFiles("index/accessory.html"))
	registrationTpl = template.Must(template.ParseFiles("index/registration.html"))
	loginTpl        = template.Must(template.ParseFiles("index/login.html"))
	adminTpl        = template.Must(template.ParseFiles("index/add_product.html"))
	orderTpl        = template.Must(template.ParseFiles("index/order.html"))
	orderSuccessTpl = template.Must(template.ParseFiles("index/order_success.html"))
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("секретный_ключ_магазина_одежды"))

func initDB() {
	var err error
	connStr := "host=localhost port=5432 user=postgres password=postgress dbname=velur sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Не удалось подключиться к БД:", err)
	}

	log.Println("Успешное подключение к базе данных")

	createClothesTable()
	createAccessoriesTable()
	createUsersTable()
	createOrdersTable()
}

func createClothesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS clothes (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		image_url VARCHAR(500),
		price DECIMAL(10, 2) NOT NULL,
		size VARCHAR(10),
		color VARCHAR(50),
		material VARCHAR(100),
		type VARCHAR(100),
		season VARCHAR(50)
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы clothes:", err)
	}
	log.Println("Таблица clothes создана/проверена")
}

func createAccessoriesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS accessories (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		image_url VARCHAR(500),
		price DECIMAL(10, 2) NOT NULL,
		type VARCHAR(100),
		color VARCHAR(50),
		material VARCHAR(100),
		target VARCHAR(100)
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы accessories:", err)
	}
	log.Println("Таблица accessories создана/проверена")
}

func createUsersTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(100) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'user'
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы users:", err)
	}
	log.Println("Таблица users создана/проверена")
}

func createOrdersTable() {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		product_name VARCHAR(255) NOT NULL,
		product_category VARCHAR(50) NOT NULL,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		middle_name VARCHAR(100),
		phone VARCHAR(20) NOT NULL,
		quantity INTEGER NOT NULL,
		region VARCHAR(100) NOT NULL,
		city VARCHAR(100) NOT NULL,
		street VARCHAR(100) NOT NULL,
		house VARCHAR(20) NOT NULL,
		apartment VARCHAR(20),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы orders:", err)
	}
	log.Println("Таблица orders создана/проверена")
}

func loadProducts() {
	rows, err := db.Query("SELECT id, name, description, image_url, price, size, color, material, type, season FROM clothes")
	if err != nil {
		log.Println("Ошибка при загрузке одежды:", err)
		return
	}
	defer rows.Close()

	products.Clothes = make(map[string]products.Clothing)

	for rows.Next() {
		var clothing products.Clothing
		if err := rows.Scan(&clothing.ID, &clothing.Name, &clothing.Description, &clothing.ImageURL, &clothing.Price,
			&clothing.Size, &clothing.Color, &clothing.Material, &clothing.Type, &clothing.Season); err != nil {
			log.Println("Ошибка при сканировании строки одежды:", err)
			continue
		}

		clothing.ImageURL = strings.Replace(clothing.ImageURL, "\\", "/", -1)
		products.Clothes[clothing.ID] = clothing
	}

	rowsAccessories, err := db.Query("SELECT id, name, description, image_url, price, type, color, material, target FROM accessories")
	if err != nil {
		log.Println("Ошибка при загрузке аксессуаров:", err)
		return
	}
	defer rowsAccessories.Close()

	products.Accessories = make(map[string]products.Accessory)

	for rowsAccessories.Next() {
		var accessory products.Accessory
		if err := rowsAccessories.Scan(&accessory.ID, &accessory.Name, &accessory.Description, &accessory.ImageURL, &accessory.Price,
			&accessory.Type, &accessory.Color, &accessory.Material, &accessory.Target); err != nil {
			log.Println("Ошибка при сканировании строки аксессуара:", err)
			continue
		}

		accessory.ImageURL = strings.Replace(accessory.ImageURL, "\\", "/", -1)
		products.Accessories[accessory.ID] = accessory
	}

	log.Printf("Загружено %d товаров одежды и %d аксессуаров", len(products.Clothes), len(products.Accessories))
}

func clothingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	clothing, exists := products.Clothes[productID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	err := clothingTpl.Execute(w, clothing)
	if err != nil {
		log.Println("Ошибка при рендеринге шаблона одежды:", err)
		http.Error(w, "Ошибка при отображении страницы", http.StatusInternalServerError)
	}
}

func accessoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	accessory, exists := products.Accessories[productID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	err := accessoryTpl.Execute(w, accessory)
	if err != nil {
		log.Println("Ошибка при рендеринге шаблона аксессуара:", err)
		http.Error(w, "Ошибка при отображении страницы", http.StatusInternalServerError)
	}
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	clothing, exists := products.Clothes[productID]
	if exists {
		err := orderTpl.Execute(w, map[string]interface{}{
			"ProductName": clothing.Name,
			"Category":    "Одежда",
			"ProductID":   productID,
		})
		if err != nil {
			log.Println("Ошибка при рендеринге шаблона заказа (одежда):", err)
			http.Error(w, "Ошибка при отображении страницы", http.StatusInternalServerError)
		}
		return
	}

	accessory, existsAccessory := products.Accessories[productID]
	if existsAccessory {
		err := orderTpl.Execute(w, map[string]interface{}{
			"ProductName": accessory.Name,
			"Category":    "Аксессуары",
			"ProductID":   productID,
		})
		if err != nil {
			log.Println("Ошибка при рендеринге шаблона заказа (аксессуар):", err)
			http.Error(w, "Ошибка при отображении страницы", http.StatusInternalServerError)
		}
		return
	}

	http.NotFound(w, r)
}

func submitOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		productName := r.FormValue("product_name")
		productCategory := r.FormValue("product_category")
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		middleName := r.FormValue("middle_name")
		phone := r.FormValue("phone")
		quantityStr := r.FormValue("quantity")
		region := r.FormValue("region")
		city := r.FormValue("city")
		street := r.FormValue("street")
		house := r.FormValue("house")
		apartment := r.FormValue("apartment")

		// Преобразуем количество
		quantity, err := strconv.Atoi(quantityStr)
		if err != nil {
			log.Println("Ошибка при преобразовании количества:", err)
			http.Error(w, "Ошибка при преобразовании количества", http.StatusBadRequest)
			return
		}
		_, err = db.Exec(`
			INSERT INTO orders (product_name, product_category, first_name, last_name, middle_name, phone, quantity, region, city, street, house, apartment) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			productName, productCategory, firstName, lastName, middleName, phone, quantity, region, city, street, house, apartment)

		if err != nil {
			log.Println("Ошибка при сохранении заказа в базу данных:", err)
			http.Error(w, "Ошибка при сохранении заказа", http.StatusInternalServerError)
			return
		}

		log.Printf("Заказ успешно оформлен: %s (%s)", productName, productCategory)
		orderSuccessTpl.Execute(w, nil)
	}
}

func addClothing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		size := r.FormValue("size")
		color := r.FormValue("color")
		material := r.FormValue("material")
		clothingType := r.FormValue("type")
		season := r.FormValue("season")

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			log.Println("Ошибка при преобразовании цены:", err)
			http.Error(w, "Ошибка при преобразовании цены", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			log.Println("Ошибка при получении файла:", err)
			http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imageDir := filepath.Join("assets", "product_images")
		if err := os.MkdirAll(imageDir, os.ModePerm); err != nil {
			log.Println("Ошибка при создании директории:", err)
			http.Error(w, "Ошибка при создании директории", http.StatusInternalServerError)
			return
		}

		imagePath := filepath.Join(imageDir, header.Filename)
		out, err := os.Create(imagePath)
		if err != nil {
			log.Println("Ошибка при сохранении файла:", err)
			http.Error(w, "Ошибка при сохранении файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			log.Println("Ошибка при записи файла:", err)
			http.Error(w, "Ошибка при записи файла", http.StatusInternalServerError)
			return
		}

		imagePath = strings.Replace(imagePath, "\\", "/", -1)

		var id int
		err = db.QueryRow(`
			INSERT INTO clothes (name, description, price, image_url, size, color, material, type, season) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
			RETURNING id`,
			name, description, price, imagePath, size, color, material, clothingType, season).Scan(&id)

		if err != nil {
			log.Println("Ошибка при добавлении одежды в базу данных:", err)
			http.Error(w, "Ошибка при добавлении товара", http.StatusInternalServerError)
			return
		}

		products.Clothes[fmt.Sprint(id)] = products.Clothing{
			ID:          fmt.Sprint(id),
			Name:        name,
			Description: description,
			Price:       price,
			ImageURL:    imagePath,
			Size:        size,
			Color:       color,
			Material:    material,
			Type:        clothingType,
			Season:      season,
		}

		log.Println("Одежда успешно добавлена:", name)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func addAccessory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		accessoryType := r.FormValue("type")
		color := r.FormValue("color")
		material := r.FormValue("material")
		target := r.FormValue("target")

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			log.Println("Ошибка при преобразовании цены:", err)
			http.Error(w, "Ошибка при преобразовании цены", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			log.Println("Ошибка при получении файла:", err)
			http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imageDir := filepath.Join("assets", "product_images")
		if err := os.MkdirAll(imageDir, os.ModePerm); err != nil {
			log.Println("Ошибка при создании директории:", err)
			http.Error(w, "Ошибка при создании директории", http.StatusInternalServerError)
			return
		}

		imagePath := filepath.Join(imageDir, header.Filename)
		out, err := os.Create(imagePath)
		if err != nil {
			log.Println("Ошибка при сохранении файла:", err)
			http.Error(w, "Ошибка при сохранении файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			log.Println("Ошибка при записи файла:", err)
			http.Error(w, "Ошибка при записи файла", http.StatusInternalServerError)
			return
		}

		imagePath = strings.Replace(imagePath, "\\", "/", -1)

		var id int
		err = db.QueryRow(`
			INSERT INTO accessories (name, description, price, image_url, type, color, material, target) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
			RETURNING id`,
			name, description, price, imagePath, accessoryType, color, material, target).Scan(&id)

		if err != nil {
			log.Println("Ошибка при добавлении аксессуара в базу данных:", err)
			http.Error(w, "Ошибка при добавлении аксессуара", http.StatusInternalServerError)
			return
		}

		products.Accessories[fmt.Sprint(id)] = products.Accessory{
			ID:          fmt.Sprint(id),
			Name:        name,
			Description: description,
			Price:       price,
			ImageURL:    imagePath,
			Type:        accessoryType,
			Color:       color,
			Material:    material,
			Target:      target,
		}

		log.Println("Аксессуар успешно добавлен:", name)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func deleteClothing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	_, err := db.Exec("DELETE FROM clothes WHERE id = $1", productID)
	if err != nil {
		log.Println("Ошибка при удалении одежды из базы данных:", err)
		http.Error(w, "Ошибка при удалении товара", http.StatusInternalServerError)
		return
	}

	delete(products.Clothes, productID)
	log.Println("Одежда успешно удалена:", productID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func deleteAccessory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	_, err := db.Exec("DELETE FROM accessories WHERE id = $1", productID)
	if err != nil {
		log.Println("Ошибка при удалении аксессуара из базы данных:", err)
		http.Error(w, "Ошибка при удалении товара", http.StatusInternalServerError)
		return
	}

	delete(products.Accessories, productID)
	log.Println("Аксессуар успешно удален:", productID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func marketHandler(w http.ResponseWriter, r *http.Request) {
	loadProducts()

	data := struct {
		Clothes     map[string]products.Clothing
		Accessories map[string]products.Accessory
		Username    string
		Role        string
	}{
		Clothes:     products.Clothes,
		Accessories: products.Accessories,
	}

	session, _ := store.Get(r, "session-name")
	if username, ok := session.Values["username"].(string); ok {
		data.Username = username
	}
	if role, ok := session.Values["role"].(string); ok {
		data.Role = role
	}

	if err := marketTpl.Execute(w, data); err != nil {
		log.Println("Ошибка при рендеринге главной страницы:", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	data := struct {
		Username string
		Role     string
	}{}

	if username, ok := session.Values["username"].(string); ok {
		data.Username = username
	}
	if role, ok := session.Values["role"].(string); ok {
		data.Role = role
	}

	aboutTpl.Execute(w, data)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Ошибка хеширования пароля", http.StatusInternalServerError)
			return
		}

		role := "user"
		if username == "admin" {
			role = "admin"
		}

		_, err = db.Exec("INSERT INTO users (username, password, email, role) VALUES ($1, $2, $3, $4)",
			username, hashedPassword, email, role)

		if err != nil {
			log.Printf("Ошибка при сохранении пользователя: %v", err)
			http.Error(w, "Ошибка при сохранении пользователя", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	registrationTpl.Execute(w, nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var hashedPassword string
		var role string
		err := db.QueryRow("SELECT password, role FROM users WHERE username = $1", username).Scan(&hashedPassword, &role)

		if err != nil || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Неверное имя пользователя или пароль",
			}
			loginTpl.Execute(w, data)
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["username"] = username
		session.Values["role"] = role
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	loginTpl.Execute(w, nil)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "username")
	delete(session.Values, "role")
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.Values["role"] != "admin" {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	loadProducts()
	data := struct {
		Clothes     map[string]products.Clothing
		Accessories map[string]products.Accessory
	}{
		Clothes:     products.Clothes,
		Accessories: products.Accessories,
	}

	adminTpl.Execute(w, data)
}

func addProductHandler(w http.ResponseWriter, r *http.Request) {
	adminTpl.Execute(w, nil)
}

func main() {
	initDB()
	loadProducts()

	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("assets"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	r.HandleFunc("/", marketHandler).Methods("GET")
	r.HandleFunc("/about", aboutHandler).Methods("GET")
	r.HandleFunc("/registration", registrationHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/admin", adminHandler).Methods("GET")
	r.HandleFunc("/admin/add-clothing", addClothing).Methods("POST")
	r.HandleFunc("/admin/add-accessory", addAccessory).Methods("POST")
	r.HandleFunc("/admin/delete-clothing/{id:[0-9]+}", deleteClothing).Methods("POST")
	r.HandleFunc("/admin/delete-accessory/{id:[0-9]+}", deleteAccessory).Methods("POST")
	r.HandleFunc("/clothing/{id:[0-9]+}", clothingHandler).Methods("GET")
	r.HandleFunc("/accessory/{id:[0-9]+}", accessoryHandler).Methods("GET")
	r.HandleFunc("/order/clothing/{id:[0-9]+}", orderHandler).Methods("GET")
	r.HandleFunc("/order/accessory/{id:[0-9]+}", orderHandler).Methods("GET")
	r.HandleFunc("/order", submitOrderHandler).Methods("POST")

	log.Println("Запуск веб-сервера магазина Velur на http://localhost:7070")
	err := http.ListenAndServe(":7070", r)
	if err != nil {
		log.Fatal("Ошибка при запуске сервера:", err)
	}
}



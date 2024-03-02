# Exploding Kitten Backend

Welcome to the backend of the Exploding Kitten game! This server handles user authentication, game logic, and communication with the database.

## Technologies Used

- Go (Golang)
- Redis (for data storage)
- JSON Web Tokens (JWT) for authentication

## Project Structure

The project is structured as follows:

- `main.go`: The main entry point for the Go server.
- `redis_ca.pem`: SSL certificate for connecting to the Redis server securely.

## Getting Started

1. Clone this repository.
2. Navigate to the `exploding-kitten-backend` directory.
3. Ensure you have Go installed on your machine.
4. Run `go run main.go` to start the backend server.
5. The server will be running on [http://localhost:8080](http://localhost:8080).

## Deployment

This backend can be deployed on platforms like Heroku, AWS, or any other service that supports Go applications.

## Database

This project uses Redis as the database. Make sure to configure your Redis server and update the connection details in `main.go`.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

package secondaryfunctions

// DBConfig holds the database configuration details
var DBConfig = struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}{
	Username: "root",
	Password: "password", // Replace with your actual password
	Host:     "127.0.0.1",
	Port:     "3306",
	Database: "students1",
}

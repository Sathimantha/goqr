CREATE TABLE errors (
    id BIGINT NOT NULL AUTO_INCREMENT,
    timestamp DATETIME NOT NULL,
    error_type VARCHAR(50) NOT NULL,
    remark TEXT,
    PRIMARY KEY (id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_error_type (error_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE students (
    student_id VARCHAR(50) PRIMARY KEY,
    full_name VARCHAR(100) NOT NULL,
    NID VARCHAR(100),
    phone_no VARCHAR(50),
    remark LONGTEXT
);ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

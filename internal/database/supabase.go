package database

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hypernova-labs/dgi-service/internal/config"
	"github.com/sirupsen/logrus"
)

// SupabaseClient representa el cliente de Supabase usando S3
type SupabaseClient struct {
	s3Client *s3.Client
	config   *config.SupabaseConfig
	logger   *logrus.Logger
	bucket   string
}

// NewSupabaseClient crea una nueva instancia del cliente de Supabase
func NewSupabaseClient(cfg *config.SupabaseConfig, logger *logrus.Logger) (*SupabaseClient, error) {
	// Crear configuración S3 personalizada para Supabase
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.StorageEndpoint,
			SigningRegion:     cfg.StorageRegion,
			HostnameImmutable: true,
		}, nil
	})

	// Crear configuración AWS
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
			},
		}),
		awsconfig.WithRegion(cfg.StorageRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS config: %w", err)
	}

	// Crear cliente S3
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // Importante para Supabase
	})

	return &SupabaseClient{
		s3Client: s3Client,
		config:   cfg,
		logger:   logger,
		bucket:   "invoice-files",
	}, nil
}

// HealthCheck verifica la conexión a Supabase
func (s *SupabaseClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar que el bucket existe
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("error checking Supabase storage connection: %w", err)
	}

	s.logger.Info("Supabase storage connection healthy")
	return nil
}

// GetClient retorna el cliente S3
func (s *SupabaseClient) GetClient() *s3.Client {
	return s.s3Client
}

// UploadFile sube un archivo al storage de Supabase
func (s *SupabaseClient) UploadFile(ctx context.Context, bucketName, fileName string, fileData []byte) (string, error) {
	// Convertir []byte a io.Reader
	reader := bytes.NewReader(fileData)
	
	// Subir archivo al bucket S3
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(fileName),
		Body:          reader,
		ContentType:   aws.String("application/octet-stream"),
		ContentLength: aws.Int64(int64(len(fileData))),
	})
	if err != nil {
		return "", fmt.Errorf("error uploading file to Supabase storage: %w", err)
	}

	// Generar URL pública
	url := fmt.Sprintf("%s/%s/%s", s.config.StorageEndpoint, bucketName, fileName)
	
	s.logger.WithFields(logrus.Fields{
		"bucket": bucketName,
		"file":   fileName,
		"url":    url,
		"size":   len(fileData),
	}).Info("File uploaded to Supabase storage successfully")

	return url, nil
}

// DownloadFile descarga un archivo del storage de Supabase
func (s *SupabaseClient) DownloadFile(ctx context.Context, bucketName, fileName string) ([]byte, error) {
	// Descargar archivo del bucket S3
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return nil, fmt.Errorf("error downloading file from Supabase storage: %w", err)
	}
	defer result.Body.Close()

	// Leer el contenido del archivo
	fileData, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading file content: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"bucket": bucketName,
		"file":   fileName,
		"size":   len(fileData),
	}).Info("File downloaded from Supabase storage successfully")

	return fileData, nil
}

// DeleteFile elimina un archivo del storage de Supabase
func (s *SupabaseClient) DeleteFile(ctx context.Context, bucketName, fileName string) error {
	// Eliminar archivo del bucket S3
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return fmt.Errorf("error deleting file from Supabase storage: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"bucket": bucketName,
		"file":   fileName,
	}).Info("File deleted from Supabase storage successfully")

	return nil
}

// CreateBucket crea un bucket en el storage de Supabase
func (s *SupabaseClient) CreateBucket(ctx context.Context, bucketName string, isPublic bool) error {
	// Crear bucket S3
	_, err := s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("error creating bucket in Supabase storage: %w", err)
	}

	// Si es público, configurar política de acceso público
	if isPublic {
		policy := `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Sid": "PublicReadGetObject",
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + bucketName + `/*"
				}
			]
		}`

		_, err = s.s3Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: aws.String(bucketName),
			Policy: aws.String(policy),
		})
		if err != nil {
			s.logger.Warnf("Could not set public policy for bucket %s: %v", bucketName, err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"bucket":  bucketName,
		"public":  isPublic,
	}).Info("Bucket created in Supabase storage successfully")

	return nil
}

// Close cierra la conexión a Supabase
func (s *SupabaseClient) Close() error {
	// El cliente S3 no tiene método Close, pero podemos limpiar recursos si es necesario
	return nil
}

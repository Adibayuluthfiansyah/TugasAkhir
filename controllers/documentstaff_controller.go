package controllers

import (
	"bytes"
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/services"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// HELPER FOR HANDLE DOCUMENT STAFF UPLOAD - REFACTOR THESIS
func handleDocumentStaffUpload(fileHeader *multipart.FileHeader, document *models.DocumentStaff) (map[string]interface{}, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("tidak dapat membuka file")
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file buffer")
	}

	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

	var resourceType string
	var folder string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar"

	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
		folder = "document_staff"

	default:
		return nil, fmt.Errorf("format file tidak didukung")
	}

	uploadResult, err := config.UploadToCloudinary(
		reader,
		fileHeader.Filename,
		folder,
		resourceType,
	)
	if err != nil {
		return nil, fmt.Errorf("upload gagal: %v", err)
	}

	if document != nil && document.PublicID != "" {
		config.DeleteFromCloudinary(
			document.PublicID,
			document.ResourceType,
		)
	}

	return map[string]interface{}{
		"file_name":     fileHeader.Filename,
		"file_url":      uploadResult.SecureURL,
		"public_id":     uploadResult.PublicID,
		"resource_type": resourceType,
	}, nil
}

// ======================================================
// CREATE STAFF DOCUMENT
// ======================================================
func CreateDocumentStaff(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userRaw.(models.User)

	subject := c.PostForm("subject")

	if subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject wajib diisi"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	// 1. BUKA FILE
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
		return
	}
	defer src.Close()

	// 2. BACA KE BUFFER
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file buffer"})
		return
	}
	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType string
	var folder string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
		folder = "document_staff"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
		return
	}

	// 3. UPLOAD KE CLOUDINARY
	uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	// 4. SIMPAN DB
	document := models.DocumentStaff{
		UserID:       user.ID,
		Subject:      subject,
		FileName:     fileHeader.Filename,
		FileURL:      uploadResult.SecureURL,
		PublicID:     uploadResult.PublicID,
		ResourceType: resourceType,
	}

	if err := config.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(uploadResult.PublicID, resourceType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error: " + err.Error()})
		return
	}

	config.DB.Preload("User").Find(&document)

	services.CreateActivity(
		user.ID,
		user.Name,
		"create",
		"Mengunggah dokumen staff: "+document.FileName,
	)

	services.NotifyAdmins(
		"Dokumen baru dari "+user.Name,
		document.FileURL,
	)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Dokumen berhasil diupload",
		"document": document,
	})
}

// ======================================================
// GET ALL STAFF DOCUMENTS - DIPERBAIKI
// ======================================================
func GetDocumentStaffs(c *gin.Context) {
	var documents []models.DocumentStaff

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	search := c.Query("search")
	userFilter := c.Query("user_id")

	query := config.DB.Model(&models.DocumentStaff{}).
		Joins("LEFT JOIN users AS User ON document_staffs.user_id = User.id")

	if search != "" {
		searchQuery := "%" + search + "%"
		query = query.Where(
			"document_staffs.subject LIKE ? OR document_staffs.file_name LIKE ? OR User.name LIKE ?",
			searchQuery, searchQuery, searchQuery,
		)
	}

	if userFilter != "" && userFilter != "all" {
		query = query.Where("document_staffs.user_id = ?", userFilter)
	}

	var total int64
	countQuery := config.DB.Model(&models.DocumentStaff{}).
		Joins("LEFT JOIN users AS User ON document_staffs.user_id = User.id")

	if search != "" {
		searchQuery := "%" + search + "%"
		countQuery = countQuery.Where(
			"document_staffs.subject LIKE ? OR document_staffs.file_name LIKE ? OR User.name LIKE ?",
			searchQuery, searchQuery, searchQuery,
		)
	}

	if userFilter != "" && userFilter != "all" {
		countQuery = countQuery.Where("document_staffs.user_id = ?", userFilter)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		total = 0
	}

	offset := (page - 1) * perPage
	err := query.
		Select("document_staffs.*").
		Offset(offset).
		Limit(perPage).
		Order("document_staffs.created_at DESC").
		Preload("User").
		Find(&documents).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil dokumen"})
		return
	}

	lastPage := 0
	if perPage > 0 {
		lastPage = int(total) / perPage
		if int(total)%perPage != 0 {
			lastPage++
		}
	}
	if lastPage == 0 && total > 0 {
		lastPage = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"documents":    documents,
		"total":        total,
		"current_page": page,
		"last_page":    lastPage,
		"per_page":     perPage,
	})
}

// ======================================================
// GET BY ID
// ======================================================
func GetDocumentStaffByID(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.Preload("User").First(&document, idQuery, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": docNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

// ======================================================
// GET PERSONAL DOCUMENTS (User's own documents only)
// ======================================================
func GetPersonalDocumentStaffs(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userRaw.(models.User)

	var documents []models.DocumentStaff

	if err := config.DB.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil dokumen pribadi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"total":     len(documents),
	})
}

// ======================================================
// UPDATE STAFF DOCUMENT
// ======================================================
func UpdateDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, idQuery, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": docNotFound})
		return
	}

	subject := c.PostForm("subject")

	updates := map[string]interface{}{}

	if subject != "" {
		updates["subject"] = subject
	}

	// REFACTOR COGNITIVE COMPLEXITY START FROM THIS

	fileHeader, err := c.FormFile("file")
	if err == nil {

		fileUpdates, err := handleDocumentStaffUpload(
			fileHeader,
			&document,
		)
		if err != nil {

			if strings.Contains(err.Error(), "format file") {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}

			return
		}

		for key, value := range fileUpdates {
			updates[key] = value
		}
	}

	// REFACTOR COGNITIVE COMPLEXITY END

	if len(updates) > 0 {
		if err := config.DB.Model(&document).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}
	}

	config.DB.Preload("User").Find(&document)

	if userRaw, ok := c.Get("user"); ok {
		user := userRaw.(models.User)
		services.CreateActivity(
			user.ID,
			user.Name,
			"update",
			"Memperbarui dokumen staff: "+document.FileName,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diperbarui",
		"document": document,
	})
}

// ======================================================
// DELETE
// ======================================================
func DeleteDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, idQuery, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": docNotFound})
		return
	}

	config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
	config.DB.Delete(&document)

	if userRaw, ok := c.Get("user"); ok {
		user := userRaw.(models.User)
		services.CreateActivity(
			user.ID,
			user.Name,
			"delete",
			"Menghapus dokumen staff: "+document.FileName,
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen berhasil dihapus"})
}

// ======================================================
// DOWNLOAD (Redirect)
// ======================================================
func DownloadDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, idQuery, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": docNotFound})
		return
	}

	if document.FileURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File tidak tersedia"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, document.FileURL)
}

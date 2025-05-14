package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"kubeants-harbor/config"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Project struct {
	Name string `json:"name"`
}

type Repository struct {
	Name string `json:"name"`
}

type Artifact struct {
	Digest string `json:"digest"`
	Tags   []Tag  `json:"tags"`
}

type Tag struct {
	Name string `json:"name"`
}

type CopyRequest struct {
	SrcProject  string `json:"src_project"`
	SrcRepo     string `json:"src_repo"`
	SrcTag      string `json:"src_tag"`
	DestProject string `json:"dest_project"`
	DestRepo    string `json:"dest_repo"`
	DestTag     string `json:"dest_tag"`
}

type CopyRepositoryRequest struct {
	SrcProject  string `json:"src_project"`
	SrcRepo     string `json:"src_repo"`
	DestProject string `json:"dest_project"`
	DestRepo    string `json:"dest_repo"`
}

func main() {
	// 加载配置
	config.LoadConfig("config.yaml")

	r := gin.Default()

	r.GET("/api/projects", handleGetProjects)
	r.GET("/api/projects/:project/repositories", handleGetRepositories)
	r.GET("/api/projects/:project/repositories/:repository/artifacts", handleGetArtifacts)
	r.POST("/api/copy-image", handleCopyImage)
	r.POST("/api/copy-repository", handleCopyRepository)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleGetProjects(c *gin.Context) {
	projects, err := getAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func handleGetRepositories(c *gin.Context) {
	project, err := url.PathUnescape(c.Param("project"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project name"})
		return
	}

	repos, err := getRepositories(project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, repos)
}

func handleGetArtifacts(c *gin.Context) {
	project, _ := url.PathUnescape(c.Param("project"))
	repository, _ := url.PathUnescape(c.Param("repository"))

	artifacts, err := getArtifacts(project, repository)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, artifacts)
}

func handleCopyImage(c *gin.Context) {
	var req CopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := copyArtifact(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image copy initiated successfully"})
}

func handleCopyRepository(c *gin.Context) {
	var req CopyRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := copyRepository(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository copy completed successfully"})
}

func getAllProjects() ([]Project, error) {
	var allProjects []Project
	for page := 1; ; page++ {
		projects, err := getPaginatedProjects(page, 100)
		if err != nil {
			return nil, err
		}
		if len(projects) == 0 {
			break
		}
		allProjects = append(allProjects, projects...)
	}
	return allProjects, nil
}

func getPaginatedProjects(page, pageSize int) ([]Project, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects?page=%d&page_size=%d",
		config.Global.Harbor.URL, page, pageSize)

	resp, err := httpRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func getRepositories(project string) ([]Repository, error) {
	var allRepos []Repository
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories?page=%d&page_size=100",
			config.Global.Harbor.URL, url.PathEscape(project), page)

		resp, err := httpRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var repos []Repository
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}
		allRepos = append(allRepos, repos...)
	}
	return allRepos, nil
}

func getArtifacts(project, repository string) ([]Artifact, error) {
	var allArtifacts []Artifact
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts?page=%d&page_size=100",
			config.Global.Harbor.URL, url.PathEscape(project), url.PathEscape(repository), page)

		resp, err := httpRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var artifacts []Artifact
		if err := json.NewDecoder(resp.Body).Decode(&artifacts); err != nil {
			return nil, err
		}

		if len(artifacts) == 0 {
			break
		}
		allArtifacts = append(allArtifacts, artifacts...)
	}
	return allArtifacts, nil
}

func copyArtifact(req CopyRequest) error {
	srcRef := fmt.Sprintf("%s/%s:%s", req.SrcProject, req.SrcRepo, req.SrcTag)
	destURL := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts?from=%s",
		config.Global.Harbor.URL,
		url.PathEscape(req.DestProject),
		url.PathEscape(req.DestRepo),
		url.QueryEscape(srcRef),
	)

	resp, err := httpRequest("POST", destURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("copy failed with status code %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func copyRepository(req CopyRepositoryRequest) error {
	artifacts, err := getArtifacts(req.SrcProject, req.SrcRepo)
	if err != nil {
		return fmt.Errorf("failed to get artifacts: %v", err)
	}

	for _, artifact := range artifacts {
		for range artifact.Tags {
			srcRef := fmt.Sprintf("%s/%s@%s", req.SrcProject, req.SrcRepo, artifact.Digest)
			destURL := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts?from=%s",
				config.Global.Harbor.URL,
				url.PathEscape(req.DestProject),
				url.PathEscape(req.DestRepo),
				url.QueryEscape(srcRef),
			)

			resp, err := httpRequest("POST", destURL, nil)
			if err != nil {
				return fmt.Errorf("copy failed for digest %s: %v", artifact.Digest, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("copy failed for digest %s: status %d, response: %s",
					artifact.Digest, resp.StatusCode, string(body))
			}
		}
	}
	return nil
}

func httpRequest(method, targetURL string, body io.Reader) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(config.Global.Harbor.Username, config.Global.Harbor.Password)
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

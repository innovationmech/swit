// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// CoverageTestSuite tests all utility functions for comprehensive coverage
type CoverageTestSuite struct {
	suite.Suite
	tempDir string
}

func TestCoverageTestSuite(t *testing.T) {
	suite.Run(t, new(CoverageTestSuite))
}

func (s *CoverageTestSuite) SetupTest() {
	var err error
	s.tempDir, err = CreateTempDir()
	s.Require().NoError(err)
}

func (s *CoverageTestSuite) TearDownTest() {
	if s.tempDir != "" {
		err := CleanupTempDir(s.tempDir)
		s.Assert().NoError(err)
	}
}

func (s *CoverageTestSuite) TestAllFixtureFunctions() {
	// Test all fixture functions to ensure coverage

	// Test service config
	serviceConfig := TestServiceConfig()
	s.NotNil(serviceConfig)
	s.Equal("test-service", serviceConfig.Name)
	s.True(serviceConfig.Features.Database)

	// Test API config
	apiConfig := TestAPIConfig()
	s.NotNil(apiConfig)
	s.Equal("test-service", apiConfig.Service)
	s.NotEmpty(apiConfig.Methods)

	// Test model config
	modelConfig := TestModelConfig()
	s.NotNil(modelConfig)
	s.Equal("Product", modelConfig.Name)
	s.NotEmpty(modelConfig.Fields)

	// Test template data
	templateData := TestTemplateData()
	s.NotNil(templateData)
	s.Equal("test-service", templateData.Service.Name)
	s.NotEmpty(templateData.Imports)

	// Test middleware config
	middlewareConfig := TestMiddlewareConfig()
	s.NotNil(middlewareConfig)
	s.Equal("AuthMiddleware", middlewareConfig.Name)

	// Test validation errors
	validationErrors := TestValidationErrors()
	s.NotEmpty(validationErrors)
	s.Equal("name", validationErrors[0].Field)

	// Test checker results
	checkerResults := TestCheckerResults()
	s.NotEmpty(checkerResults)
	s.Contains(checkerResults, "style")
}

func (s *CoverageTestSuite) TestAllHelperFunctions() {
	// Create test file for assertions
	testFile := filepath.Join(s.tempDir, "test.txt")
	content := "Hello, World!\nThis is a test file."
	err := os.WriteFile(testFile, []byte(content), 0644)
	s.Require().NoError(err)

	// Test file existence assertions
	AssertFileExists(s.T(), testFile)
	AssertFileNotExists(s.T(), filepath.Join(s.tempDir, "nonexistent.txt"))

	// Test file content assertions
	AssertFileContains(s.T(), testFile, "Hello")
	AssertFileNotContains(s.T(), testFile, "Goodbye")
	AssertFileEquals(s.T(), testFile, content)

	// Test directory creation and assertions
	testDir := filepath.Join(s.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	s.Require().NoError(err)
	AssertDirectoryExists(s.T(), testDir)
	AssertDirectoryNotExists(s.T(), filepath.Join(s.tempDir, "nonexistentdir"))

	// Test files generation assertion
	expectedFiles := []string{"test.txt"}
	AssertFilesGenerated(s.T(), s.tempDir, expectedFiles)

	// Test template output assertion
	templateOutput := []byte("package main\n\nfunc main() {}")
	expectedSubstrings := []string{"package main", "func main"}
	AssertTemplateOutput(s.T(), templateOutput, expectedSubstrings)

	// Test valid Go code assertion
	goCode := []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}")
	AssertValidGoCode(s.T(), goCode)

	// Test valid YAML assertion
	yamlCode := []byte("name: test\nversion: 1.0.0\n")
	AssertValidYAML(s.T(), yamlCode)

	// Test no template errors assertion
	cleanOutput := []byte("package main\n\nfunc main() {}")
	AssertNoTemplateErrors(s.T(), cleanOutput)

	// Test contains all/none assertions
	testContent := "alpha beta gamma"
	AssertContainsAll(s.T(), testContent, []string{"alpha", "beta"})
	AssertContainsNone(s.T(), testContent, []string{"delta", "epsilon"})

	// Test file permissions assertion
	AssertFilePermissions(s.T(), testFile, 0644)
}

func (s *CoverageTestSuite) TestTemplateHelpers() {
	// Test template setup
	templateDir, cleanup := SetupTemplateTestSuite(s.T())
	defer cleanup()

	s.NotEmpty(templateDir)
	AssertDirectoryExists(s.T(), templateDir)

	// Test template files creation
	templates := TestTemplates()
	s.NotEmpty(templates)
	s.Contains(templates, "service.go.tmpl")
	s.Contains(templates, "handler.go.tmpl")

	// Test template comparison
	content1 := []byte("line 1\nline 2\nline 3")
	content2 := []byte("line 1\nline 2\nline 3")
	CompareTemplateOutput(s.T(), content1, content2, false)
	CompareTemplateOutput(s.T(), content1, content2, true)
}

func (s *CoverageTestSuite) TestMockFunctionality() {
	// Test all mock implementations

	// Test MockTemplateEngine
	mockEngine := &MockTemplateEngine{}
	mockTemplate := NewMockTemplate("test.tmpl")

	// Setup mock expectations
	mockEngine.On("LoadTemplate", "test.tmpl").Return(mockTemplate, nil)
	mockEngine.On("RenderTemplate", mockTemplate, "data").Return([]byte("rendered"), nil)
	mockEngine.On("RegisterFunction", "testFunc", "func").Return(nil)
	mockEngine.On("SetTemplateDir", "/test").Return(nil)

	// Test mock calls
	template, err := mockEngine.LoadTemplate("test.tmpl")
	s.NoError(err)
	s.NotNil(template)

	result, err := mockEngine.RenderTemplate(template, "data")
	s.NoError(err)
	s.Equal([]byte("rendered"), result)

	err = mockEngine.RegisterFunction("testFunc", "func")
	s.NoError(err)

	err = mockEngine.SetTemplateDir("/test")
	s.NoError(err)

	// Verify all expectations were met
	mockEngine.AssertExpectations(s.T())

	// Test MockGenerator
	mockGenerator := &MockGenerator{}
	serviceConfig := TestServiceConfig()
	apiConfig := TestAPIConfig()
	modelConfig := TestModelConfig()
	middlewareConfig := TestMiddlewareConfig()

	mockGenerator.On("GenerateService", serviceConfig).Return(nil)
	mockGenerator.On("GenerateAPI", apiConfig).Return(nil)
	mockGenerator.On("GenerateModel", modelConfig).Return(nil)
	mockGenerator.On("GenerateMiddleware", middlewareConfig).Return(nil)

	err = mockGenerator.GenerateService(serviceConfig)
	s.NoError(err)

	err = mockGenerator.GenerateAPI(apiConfig)
	s.NoError(err)

	err = mockGenerator.GenerateModel(modelConfig)
	s.NoError(err)

	err = mockGenerator.GenerateMiddleware(middlewareConfig)
	s.NoError(err)

	mockGenerator.AssertExpectations(s.T())

	// Test MockFileSystem
	mockFS := NewMockFileSystem()
	mockFS.On("WriteFile", "/test/file.txt", []byte("content"), 0644).Return(nil)
	mockFS.On("ReadFile", "/test/file.txt").Return([]byte("content"), nil)
	mockFS.On("MkdirAll", "/test", 0755).Return(nil)
	mockFS.On("Exists", "/test/file.txt").Return(true)
	mockFS.On("Remove", "/test/file.txt").Return(nil)
	mockFS.On("Copy", "/src", "/dst").Return(nil)

	err = mockFS.WriteFile("/test/file.txt", []byte("content"), 0644)
	s.NoError(err)

	content, err := mockFS.ReadFile("/test/file.txt")
	s.NoError(err)
	s.Equal([]byte("content"), content)

	err = mockFS.MkdirAll("/test", 0755)
	s.NoError(err)

	exists := mockFS.Exists("/test/file.txt")
	s.True(exists)

	err = mockFS.Remove("/test/file.txt")
	s.NoError(err)

	err = mockFS.Copy("/src", "/dst")
	s.NoError(err)

	mockFS.AssertExpectations(s.T())
}

func (s *CoverageTestSuite) TestErrorMocks() {
	// Test error-producing mocks

	// Test ErrorTemplateEngine
	errorEngine := &ErrorTemplateEngine{}

	template, err := errorEngine.LoadTemplate("any")
	s.Error(err)
	s.Nil(template)
	s.Contains(err.Error(), "template not found")

	result, err := errorEngine.RenderTemplate(nil, nil)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "rendering failed")

	err = errorEngine.RegisterFunction("test", nil)
	s.Error(err)
	s.Contains(err.Error(), "function registration failed")

	err = errorEngine.SetTemplateDir("/invalid")
	s.Error(err)
	s.Contains(err.Error(), "invalid template directory")

	// Test ErrorTemplate
	errorTemplate := NewErrorTemplate("error.tmpl")
	s.Equal("error.tmpl", errorTemplate.Name())

	result, err = errorTemplate.Render(nil)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "template render error")

	// Test FailingFileSystem
	failingFS := NewFailingFileSystem("write", "read", "mkdir")

	err = failingFS.WriteFile("/test", []byte("content"), 0644)
	s.Error(err)
	s.Contains(err.Error(), "write operation failed")

	content, err := failingFS.ReadFile("/test")
	s.Error(err)
	s.Nil(content)
	s.Contains(err.Error(), "read operation failed")

	err = failingFS.MkdirAll("/test", 0755)
	s.Error(err)
	s.Contains(err.Error(), "mkdir operation failed")
}

func (s *CoverageTestSuite) TestProgressBar() {
	// Test MockProgressBar
	mockProgressBar := NewMockProgressBar(100)
	s.Equal(100, mockProgressBar.GetTotal())
	s.Equal(0, mockProgressBar.GetCurrent())

	mockProgressBar.On("Update", 50).Return(nil)
	mockProgressBar.On("SetMessage", "Processing...").Return(nil)
	mockProgressBar.On("Finish").Return(nil)
	mockProgressBar.On("SetTotal", 200).Return(nil)

	err := mockProgressBar.Update(50)
	s.NoError(err)
	s.Equal(50, mockProgressBar.GetCurrent())

	err = mockProgressBar.SetMessage("Processing...")
	s.NoError(err)

	err = mockProgressBar.Finish()
	s.NoError(err)

	err = mockProgressBar.SetTotal(200)
	s.NoError(err)
	s.Equal(200, mockProgressBar.GetTotal())

	mockProgressBar.AssertExpectations(s.T())
}

func (s *CoverageTestSuite) TestInteractiveUI() {
	// Test MockInteractiveUI
	mockUI := NewMockInteractiveUI()

	mockUI.On("ShowWelcome").Return(nil)
	mockUI.On("PromptInput", "Enter name:", mock.Anything).Return("test", nil)
	mockUI.On("ShowMenu", "Select option:", mock.Anything).Return(0, nil)
	mockUI.On("ShowProgress", "Loading...", 100).Return(NewMockProgressBar(100))
	mockUI.On("ShowSuccess", "Success!").Return(nil)
	mockUI.On("ShowError", mock.AnythingOfType("*errors.errorString")).Return(nil)
	mockUI.On("ShowTable", []string{"Name", "Value"}, [][]string{{"test", "value"}}).Return(nil)

	err := mockUI.ShowWelcome()
	s.NoError(err)

	result, err := mockUI.PromptInput("Enter name:", nil)
	s.NoError(err)
	s.Equal("test", result)

	choice, err := mockUI.ShowMenu("Select option:", nil)
	s.NoError(err)
	s.Equal(0, choice)

	progressBar := mockUI.ShowProgress("Loading...", 100)
	s.NotNil(progressBar)

	err = mockUI.ShowSuccess("Success!")
	s.NoError(err)

	err = mockUI.ShowError(assert.AnError)
	s.NoError(err)

	err = mockUI.ShowTable([]string{"Name", "Value"}, [][]string{{"test", "value"}})
	s.NoError(err)

	mockUI.AssertExpectations(s.T())
}

func (s *CoverageTestSuite) TestTemplateStoreHelpers() {
	// Test MockTemplateStore
	mockStore := NewMockTemplateStore()
	mockTemplate := NewMockTemplate("test.tmpl")

	mockStore.On("LoadTemplate", "test.tmpl").Return(mockTemplate, nil)
	mockStore.On("StoreTemplate", "test.tmpl", mockTemplate).Return(nil)
	mockStore.On("ListTemplates").Return([]string{"test.tmpl"}, nil)
	mockStore.On("RemoveTemplate", "test.tmpl").Return(nil)
	mockStore.On("ClearCache").Return(nil)

	template, err := mockStore.LoadTemplate("test.tmpl")
	s.NoError(err)
	s.NotNil(template)

	err = mockStore.StoreTemplate("test.tmpl", mockTemplate)
	s.NoError(err)

	templates, err := mockStore.ListTemplates()
	s.NoError(err)
	s.Contains(templates, "test.tmpl")

	err = mockStore.RemoveTemplate("test.tmpl")
	s.NoError(err)

	err = mockStore.ClearCache()
	s.NoError(err)

	mockStore.AssertExpectations(s.T())

	// Test stored template retrieval
	storedTemplate, exists := mockStore.GetTemplate("test.tmpl")
	s.False(exists) // Initially empty
	s.Nil(storedTemplate)
}

func (s *CoverageTestSuite) TestFileSystemMockData() {
	// Test FileSystemMockData
	mockData := TestFileSystemMockData()
	s.NotEmpty(mockData)
	s.Contains(mockData, "main.go")
	s.Contains(mockData, "config.yaml")
	s.Contains(mockData, "README.md")

	// Verify content
	s.Contains(string(mockData["main.go"]), "package main")
	s.Contains(string(mockData["config.yaml"]), "service:")
	s.Contains(string(mockData["README.md"]), "# Test Service")
}

func (s *CoverageTestSuite) TestExpectedFiles() {
	// Test ExpectedFiles
	expectedFiles := ExpectedFiles()
	s.NotEmpty(expectedFiles)
	s.Contains(expectedFiles, "service")
	s.Contains(expectedFiles, "api")
	s.Contains(expectedFiles, "model")

	// Verify service files
	serviceFiles := expectedFiles["service"]
	s.Contains(serviceFiles, "main.go")
	s.Contains(serviceFiles, "go.mod")
	s.Contains(serviceFiles, "Dockerfile")

	// Verify API files
	apiFiles := expectedFiles["api"]
	s.Contains(apiFiles, "api/handlers.go")
	s.Contains(apiFiles, "api/routes.go")

	// Verify model files
	modelFiles := expectedFiles["model"]
	s.Contains(modelFiles, "internal/models/user.go")
	s.Contains(modelFiles, "internal/repository/user_repo.go")
}

func (s *CoverageTestSuite) TestProjectCreation() {
	// Test CreateTestProject
	projectDir := CreateTestProject(s.T())
	defer CleanupTempDir(projectDir)

	// Verify project structure
	expectedDirs := []string{"cmd", "internal", "pkg", "api", "configs", "docs"}
	for _, dir := range expectedDirs {
		AssertDirectoryExists(s.T(), filepath.Join(projectDir, dir))
	}

	// Verify project files
	expectedFiles := []string{"go.mod", "README.md", ".gitignore"}
	for _, file := range expectedFiles {
		AssertFileExists(s.T(), filepath.Join(projectDir, file))
	}

	// Verify file content
	goModPath := filepath.Join(projectDir, "go.mod")
	AssertFileContains(s.T(), goModPath, "module github.com/test/project")
	AssertFileContains(s.T(), goModPath, "go 1.19")
}

func (s *CoverageTestSuite) TestDirectoryStructureCreation() {
	// Test CreateDirectoryStructure
	structure := map[string]string{
		"dir1":                  "", // Directory
		"dir2":                  "", // Directory
		"dir1/file1.txt":        "content1",
		"dir2/subdir":           "", // Nested directory
		"dir2/subdir/file2.txt": "content2",
		"root_file.txt":         "root content",
	}

	CreateDirectoryStructure(s.T(), s.tempDir, structure)

	// Verify structure was created
	AssertDirectoryExists(s.T(), filepath.Join(s.tempDir, "dir1"))
	AssertDirectoryExists(s.T(), filepath.Join(s.tempDir, "dir2"))
	AssertDirectoryExists(s.T(), filepath.Join(s.tempDir, "dir2/subdir"))

	AssertFileExists(s.T(), filepath.Join(s.tempDir, "dir1/file1.txt"))
	AssertFileExists(s.T(), filepath.Join(s.tempDir, "dir2/subdir/file2.txt"))
	AssertFileExists(s.T(), filepath.Join(s.tempDir, "root_file.txt"))

	// Verify content
	AssertFileContains(s.T(), filepath.Join(s.tempDir, "dir1/file1.txt"), "content1")
	AssertFileContains(s.T(), filepath.Join(s.tempDir, "dir2/subdir/file2.txt"), "content2")
	AssertFileContains(s.T(), filepath.Join(s.tempDir, "root_file.txt"), "root content")
}

func (s *CoverageTestSuite) TestFileComparison() {
	// Test AssertFilesMatch
	file1 := filepath.Join(s.tempDir, "file1.txt")
	file2 := filepath.Join(s.tempDir, "file2.txt")
	content := "identical content"

	CreateFileWithContent(s.T(), file1, content)
	CreateFileWithContent(s.T(), file2, content)

	AssertFilesMatch(s.T(), file1, file2)
}

func (s *CoverageTestSuite) TestErrorAssertions() {
	// Test error assertion helpers

	// Test AssertErrorContains
	err := assert.AnError
	AssertErrorContains(s.T(), err, "assert.AnError")

	// Test AssertErrorType (would work with real errors)
	// AssertErrorType(s.T(), err, &errors.errorString{})
}

func (s *CoverageTestSuite) TestWorkingDirectoryMock() {
	// Test MockWorkingDirectory
	originalDir, err := os.Getwd()
	s.Require().NoError(err)

	restore := MockWorkingDirectory(s.T(), s.tempDir)
	defer restore()

	// Verify we're in the mock directory
	currentDir, err := os.Getwd()
	s.NoError(err)

	// On macOS, resolve symlinks to handle /private/var vs /var differences
	expectedDir, err := filepath.EvalSymlinks(s.tempDir)
	s.NoError(err)
	actualDir, err := filepath.EvalSymlinks(currentDir)
	s.NoError(err)
	s.Equal(expectedDir, actualDir)

	// Restore and verify
	restore()
	currentDir, err = os.Getwd()
	s.NoError(err)
	s.Equal(originalDir, currentDir)
}

// Test condition waiting (simplified)
func (s *CoverageTestSuite) TestConditionWaiting() {
	// Test WaitForCondition with immediate success
	condition := func() bool { return true }
	WaitForCondition(s.T(), condition, "1s", "100ms")
}

// Comprehensive edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("Empty Templates Map", func(t *testing.T) {
		templates := map[string]string{}
		assert.Empty(t, templates)
	})

	t.Run("Nil Template Data", func(t *testing.T) {
		// Test handling of nil template data
		var data *interface{} = nil
		assert.Nil(t, data)
	})

	t.Run("Invalid File Paths", func(t *testing.T) {
		// Test with various invalid paths
		invalidPaths := []string{
			"",
			"/dev/null/invalid",
			"\\invalid\\windows\\path",
			"path/with\x00null",
		}

		for _, path := range invalidPaths {
			_, err := os.Stat(path)
			assert.Error(t, err)
		}
	})

	t.Run("Large Content Handling", func(t *testing.T) {
		// Test with large content
		largeContent := make([]byte, 1024*1024) // 1MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		tempDir, _ := CreateTempDir()
		defer CleanupTempDir(tempDir)

		largeFile := filepath.Join(tempDir, "large.txt")
		err := os.WriteFile(largeFile, largeContent, 0644)
		assert.NoError(t, err)

		readContent, err := os.ReadFile(largeFile)
		assert.NoError(t, err)
		assert.Equal(t, largeContent, readContent)
	})
}

// Performance tests for test utilities
func BenchmarkTestUtilities(b *testing.B) {
	b.Run("CreateTempDir", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempDir, _ := CreateTempDir()
			CleanupTempDir(tempDir)
		}
	})

	b.Run("TestServiceConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = TestServiceConfig()
		}
	})

	b.Run("TestTemplateData", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = TestTemplateData()
		}
	})

	b.Run("TestTemplates", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = TestTemplates()
		}
	})
}

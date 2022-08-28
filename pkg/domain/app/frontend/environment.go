package frontend

import "os"

const envFile = buildDir + "/env.js"

// setFrontendEnv creates the env.js file in the frontend build. This is used
// to set variables at server run time
func setFrontendEnv(apiUrl string) error {
	// remove any old environment
	_ = os.Remove(envFile)

	// content of new environment file
	envFileContent := `
	window.env = {
		API_URL: '` + apiUrl + `',
	};
	`

	file, err := os.Create(envFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(envFileContent))
	if err != nil {
		return err
	}

	return nil
}
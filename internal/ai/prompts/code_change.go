package prompts

import "fmt"

func GetSiteCodeChangePrompt(userQuery string, contextFiles string) (string, string) {
	prompt := `
		User's instruction:
		---
		%s
		---

		Here are the most relevant existing files from the project:
		---
		%s
		---

		Please respond with updated or new files in the following format:
		` + "```json" + `
		[
		{
			"filename": "src/components/Hero.tsx",
			"type": "tsx",
			"content": "..."
		},
		{
			"filename": "src/components/Testimonials.tsx",
			"type": "tsx",
			"content": "..."
		}
		]
		` + "```" + `

		Only return the modified or newly added files. Do not include duplicates or files that were not changed.
	`

	fullprompt := fmt.Sprintf(prompt, userQuery, contextFiles)
	ragSystemPrompt := `
		You are a code assistant helping to **update an existing project**. 
		Respond ONLY with the JSON array containing modified or new files as requested.
	`

	return fullprompt, ragSystemPrompt
}

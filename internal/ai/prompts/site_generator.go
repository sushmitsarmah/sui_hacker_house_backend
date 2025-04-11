package prompts

// Constant for the initial generation prompt template
func GetSiteGenerationPrompt() string {
	return `
		You are a full-stack site generator AI.

		A user has submitted the following project description:

		---
		"%s"
		---

		Please create a **multi-file project** based on the following rules:

		1.  **Frontend Framework**: React + TypeScript (Vite)
		2.  **Styling**: TailwindCSS, consistent color theme:
			*   Primary: #1A73E8
			*   Accent: #FF6F61
			*   Background: #F9FAFB
			*   Font: Inter, sans-serif
		3.  **Layout**: Responsive grid, cards with soft shadows and rounded corners
		4.  **Animations**: Use Framer Motion for subtle entry effects on buttons, cards, and modals
		5.  **Pages to Include** (at minimum):
			*   ` + "`index.tsx`" + `: landing page with hero section, feature highlights
			*   ` + "`about.tsx`" + `: about the site/project
			*   ` + "`components/Navbar.tsx`" + `, ` + "`Footer.tsx`" + `
			*   ` + "`App.tsx`" + `: wrap routes and layout
			*   ` + "`main.tsx`" + `: app root
			*   ` + "`tailwind.config.ts`" + `: theme customization
			*   ` + "`vite.config.ts`" + `: default Vite config
			*   ` + "`package.json`" + `: default package json for all libraries and dependencies

		package.json should include all the libraries used in all the files including vite.config.ts and tailwind.config.ts.
		include @vitejs/plugin-react and tailwindcss as dev dependencies.

		Respond with a structured array of files in the following format:

		` + "```json" + `
		[
		{
			"filename": "src/App.tsx",
			"type": "tsx",
			"content": "..."
		},
		{
			"filename": "src/components/Navbar.tsx",
			"type": "tsx",
			"content": "..."
		},
		...
		]
		` + "```" + `

		Only include code — no extra explanation. Your output will be parsed and saved as project files.
	`
}

import { useTheme } from "./theme-provider"

export function ThemeToggle() {
    const { theme, setTheme } = useTheme()

    return (
        <div className="flex items-center space-x-2">
            <button
                onClick={() => setTheme("light")}
                className={`px-3 py-1 rounded ${theme === "light" ? "bg-primary text-primary-foreground" : "bg-muted"}`}
            >
                Light
            </button>
            <button
                onClick={() => setTheme("dark")}
                className={`px-3 py-1 rounded ${theme === "dark" ? "bg-primary text-primary-foreground" : "bg-muted"}`}
            >
                Dark
            </button>
            <button
                onClick={() => setTheme("system")}
                className={`px-3 py-1 rounded ${theme === "system" ? "bg-primary text-primary-foreground" : "bg-muted"}`}
            >
                System
            </button>
        </div>
    )
}

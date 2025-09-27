import { useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import './App.css'
import { ThemeProvider } from './components/theme-provider'
import { ThemeToggle } from './components/theme-toggle'

function App() {
    const [count, setCount] = useState(0)

    return (
        <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
            <div className="min-h-screen bg-background text-foreground p-4">
                <header className="flex justify-between items-center mb-8">
                    <h1 className="text-2xl font-bold">RapidFlow</h1>
                    <ThemeToggle />
                </header>
                <main className="max-w-4xl mx-auto">
                    <div className="flex flex-col items-center space-y-8">
                        <div className="flex space-x-8">
                            <a href="https://vite.dev" target="_blank" className="hover:opacity-75 transition-opacity">
                                <img src={viteLogo} className="h-16 w-16" alt="Vite logo" />
                            </a>
                            <a href="https://react.dev" target="_blank" className="hover:opacity-75 transition-opacity">
                                <img src={reactLogo} className="h-16 w-16" alt="React logo" />
                            </a>
                        </div>
                        <h1 className="text-4xl font-bold text-center">Vite + React + Tailwind</h1>
                        <div className="bg-card p-6 rounded-lg shadow-lg border max-w-md">
                            <button
                                onClick={() => setCount((count) => count + 1)}
                                className="bg-primary text-primary-foreground px-4 py-2 rounded hover:bg-primary/90 transition-colors w-full"
                            >
                                count is {count}
                            </button>
                            <p className="mt-4 text-muted-foreground text-center">
                                Edit <code className="bg-muted px-1 py-0.5 rounded text-sm">src/App.tsx</code> and save to test HMR
                            </p>
                        </div>
                        <p className="text-muted-foreground text-center max-w-md">
                            Click on the Vite and React logos to learn more
                        </p>
                    </div>
                </main>
            </div>
        </ThemeProvider>
    )
}

export default App

import React from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './main.ts' // Import types
import App from './App'
import { ErrorBoundary } from './components/ErrorBoundary'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ErrorBoundary title="Postwoman could not render this view">
            <App />
        </ErrorBoundary>
    </React.StrictMode>
)
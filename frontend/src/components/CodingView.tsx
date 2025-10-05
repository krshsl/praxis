import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Editor } from '@monaco-editor/react'
import { Button } from 'components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from 'components/ui/Card'
import { Modal } from 'components/ui/Modal'
import { useConversationStore } from 'store/useStore'
import { websocketService } from 'services/websocket'

const LANGUAGES = [
  { value: 'javascript', label: 'JavaScript' },
  { value: 'python', label: 'Python' },
  { value: 'java', label: 'Java' },
  { value: 'cpp', label: 'C++' },
  { value: 'csharp', label: 'C#' },
  { value: 'go', label: 'Go' },
  { value: 'rust', label: 'Rust' },
  { value: 'typescript', label: 'TypeScript' },
]

const DEFAULT_CODE = {
  javascript: `function fibonacci(n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}`,
  python: `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)`,
  java: `public class Fibonacci {
    public static int fibonacci(int n) {
        if (n <= 1) return n;
        return fibonacci(n - 1) + fibonacci(n - 2);
    }
}`,
  cpp: `#include <iostream>
using namespace std;

int fibonacci(int n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}`,
  csharp: `public class Fibonacci {
    public static int Fibonacci(int n) {
        if (n <= 1) return n;
        return Fibonacci(n - 1) + Fibonacci(n - 2);
    }
}`,
  go: `package main

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`,
  rust: `fn fibonacci(n: u32) -> u32 {
    if n <= 1 {
        n
    } else {
        fibonacci(n - 1) + fibonacci(n - 2)
    }
}`,
  typescript: `function fibonacci(n: number): number {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}`,
}

const CODING_QUESTIONS = [
  {
    id: 1,
    title: "Two Sum",
    difficulty: "Easy",
    description: "Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.",
    example: "Input: nums = [2,7,11,15], target = 9\nOutput: [0,1]\nExplanation: Because nums[0] + nums[1] == 9, we return [0, 1]."
  },
  {
    id: 2,
    title: "Valid Parentheses",
    difficulty: "Easy",
    description: "Given a string s containing just the characters '(', ')', '{', '}', '[' and ']', determine if the input string is valid.",
    example: "Input: s = \"()\"\nOutput: true"
  },
  {
    id: 3,
    title: "Merge Two Sorted Lists",
    difficulty: "Easy",
    description: "Merge two sorted linked lists and return it as a sorted list.",
    example: "Input: l1 = [1,2,4], l2 = [1,3,4]\nOutput: [1,1,2,3,4,4]"
  }
]

export function CodingView() {
  const navigate = useNavigate()
  const [code, setCode] = useState(DEFAULT_CODE.python)
  const [language, setLanguage] = useState('python')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showEndModal, setShowEndModal] = useState(false)
  const [isEnding, setIsEnding] = useState(false)
  const [currentQuestion] = useState(CODING_QUESTIONS[0])
  const [aiFeedback] = useState('')
  const editorRef = useRef<any>(null)
  const { isConnected, messages } = useConversationStore()

  const handleEditorDidMount = (editor: any) => {
    editorRef.current = editor
  }

  const handleLanguageChange = (newLanguage: string) => {
    setLanguage(newLanguage)
    setCode(DEFAULT_CODE[newLanguage as keyof typeof DEFAULT_CODE] || '')
  }

  const handleSubmit = async () => {
    if (!isConnected) {
      alert('Not connected to server')
      return
    }

    setIsSubmitting(true)
    try {
      websocketService.sendMessage({
        type: 'code',
        content: code,
        language,
        session_id: useConversationStore.getState().currentSession || undefined
      })
    } catch (error) {
      console.error('Failed to submit code:', error)
      alert('Failed to submit code')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleFormat = () => {
    if (editorRef.current) {
      editorRef.current.getAction('editor.action.formatDocument')?.run()
    }
  }

  const handleClear = () => {
    setCode('')
  }

  const handleReset = () => {
    setCode(DEFAULT_CODE[language as keyof typeof DEFAULT_CODE] || '')
  }

  const handleEndAssessment = async () => {
    setIsEnding(true)
    try {
      await websocketService.sendMessage({
        type: 'end_session',
        session_id: useConversationStore.getState().currentSession || undefined
      })
      navigate('/dashboard')
    } catch (error) {
      console.error('Failed to end assessment:', error)
    } finally {
      setIsEnding(false)
      setShowEndModal(false)
    }
  }

  // Get latest AI feedback from messages
  const latestAiMessage = messages.filter(m => m.role === 'assistant').pop()
  const feedback = latestAiMessage?.content || aiFeedback

  return (
  <div className="h-screen flex flex-col bg-background">
      {/* Header */}
  <div className="bg-card border-b border-border px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <h1 className="text-xl font-semibold">Coding Assessment</h1>
            <div className="flex items-center space-x-2">
              <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-primary' : 'bg-destructive'}`} />
              <span className="text-sm text-muted-foreground">
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>
          <Button
            onClick={() => setShowEndModal(true)}
            variant="destructive"
            size="sm"
          >
            End Assessment
          </Button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex">
        {/* Left Side - AI Feedback (1/4) */}
        <div className="w-1/4 p-6 flex flex-col">
          <Card className="flex-1">
            <CardHeader>
              <CardTitle>AI Feedback</CardTitle>
            </CardHeader>
            <CardContent className="flex-1 flex flex-col">
              {/* AI Avatar */}
              <div className="w-16 h-16 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-xl font-bold mx-auto mb-4">
                AI
              </div>
              
              {/* AI Feedback */}
              <div className="flex-1 bg-muted rounded-lg p-4 overflow-y-auto">
                {feedback ? (
                  <div className="text-sm text-foreground whitespace-pre-wrap">
                    {feedback}
                  </div>
                ) : (
                  <div className="text-sm text-muted-foreground text-center">
                    Submit your code to get AI feedback
                  </div>
                )}
              </div>

              {/* Question Info */}
              <div className="mt-4 space-y-2">
                <h3 className="font-semibold text-sm">{currentQuestion.title}</h3>
                <span className={`px-2 py-1 rounded text-xs ${
                  currentQuestion.difficulty === 'Easy' ? 'bg-accent text-accent-foreground' :
                  currentQuestion.difficulty === 'Medium' ? 'bg-orange-3 text-orange-11' :
                  'bg-destructive/10 text-destructive-foreground'
                }`}>
                  {currentQuestion.difficulty}
                </span>
                <p className="text-xs text-muted-foreground line-clamp-3">
                  {currentQuestion.description}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Side - Code Editor (3/4) */}
        <div className="w-3/4 p-6 flex flex-col">
          <Card className="flex-1">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Code Editor</CardTitle>
                <div className="flex items-center space-x-4">
                  <select
                    value={language}
                    onChange={(e) => handleLanguageChange(e.target.value)}
                    className="rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {LANGUAGES.map((lang) => (
                      <option key={lang.value} value={lang.value}>
                        {lang.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex-1 flex flex-col">
              {/* Code Editor */}
              <div className="flex-1 border rounded-lg overflow-hidden" style={{ minHeight: '40vh', maxHeight: '60vh', height: '100%' }}>
                <Editor
                  height="100vh"
                  language={language}
                  value={code}
                  onChange={(value) => setCode(value || '')}
                  onMount={handleEditorDidMount}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 16,
                    lineNumbers: 'on',
                    roundedSelection: false,
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                    insertSpaces: true,
                    wordWrap: 'on',
                  }}
                />
              </div>

              {/* Action Buttons */}
              <div className="flex flex-wrap gap-2 mt-4">
                <Button
                  onClick={handleSubmit}
                  disabled={!isConnected || isSubmitting || !code.trim()}
                  size="lg"
                >
                  {isSubmitting ? 'Submitting...' : 'Submit Code'}
                </Button>
                
                <Button
                  onClick={handleFormat}
                  variant="outline"
                  disabled={!code.trim()}
                >
                  Format
                </Button>
                
                <Button
                  onClick={handleClear}
                  variant="outline"
                  disabled={!code.trim()}
                >
                  Clear
                </Button>
                
                <Button
                  onClick={handleReset}
                  variant="outline"
                >
                  Reset
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* End Assessment Modal */}
      <Modal
        isOpen={showEndModal}
        onClose={() => setShowEndModal(false)}
        title="End Coding Assessment"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            Are you sure you want to end this coding assessment? Your progress will be saved and you can view the results later.
          </p>
          <div className="flex space-x-3">
            <Button
              onClick={handleEndAssessment}
              disabled={isEnding}
              variant="destructive"
              className="flex-1"
            >
              {isEnding ? 'Ending...' : 'End Assessment'}
            </Button>
            <Button
              onClick={() => setShowEndModal(false)}
              variant="outline"
              className="flex-1"
            >
              Cancel
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

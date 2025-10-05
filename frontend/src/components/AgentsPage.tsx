import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from 'services/api'
import type { Agent } from 'services/api'
import { Button } from 'components/ui/Button'
import { Modal } from 'components/ui/Modal'
import { SearchableTable } from 'components/ui/SearchableTable'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'components/ui/Select'

export function AgentsPage() {
  const navigate = useNavigate()
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null)
  const [deletingAgent, setDeletingAgent] = useState<Agent | null>(null)
  const [startingAgentId, setStartingAgentId] = useState<string | null>(null)

  useEffect(() => {
    loadAgents()
  }, [])

  const loadAgents = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await apiService.getAgents()
      setAgents(response.agents)
    } catch (err) {
      setError('Failed to load agents')
      console.error('Agents load error:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleAddAgent = async (agentData: Partial<Agent>) => {
    try {
      // In a real app, you'd call an API to create the agent
      const newAgent: Agent = {
        id: Date.now().toString(),
        name: agentData.name || 'New Agent',
        description: agentData.description || '',
        personality: agentData.personality || '',
        industry: agentData.industry || '',
        level: agentData.level || '',
        is_public: agentData.is_public || false,
        is_active: agentData.is_active || true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
      setAgents(prev => [...prev, newAgent])
      setShowAddModal(false)
    } catch (err) {
      console.error('Failed to add agent:', err)
    }
  }

  const handleEditAgent = async (agentData: Partial<Agent>) => {
    try {
      // In a real app, you'd call an API to update the agent
      setAgents(prev => prev.map(agent => 
        agent.id === editingAgent?.id 
          ? { ...agent, ...agentData, updated_at: new Date().toISOString() }
          : agent
      ))
      setShowEditModal(false)
      setEditingAgent(null)
    } catch (err) {
      console.error('Failed to edit agent:', err)
    }
  }

  const handleDeleteAgent = (agent: Agent) => {
    setDeletingAgent(agent)
    setShowDeleteModal(true)
  }

  const confirmDeleteAgent = async () => {
    if (!deletingAgent) return
    
    try {
      await apiService.deleteAgent(deletingAgent.id)
      setAgents(prev => prev.filter(agent => agent.id !== deletingAgent.id))
      setShowDeleteModal(false)
      setDeletingAgent(null)
    } catch (err) {
      console.error('Failed to delete agent:', err)
      setError('Failed to delete agent')
    }
  }

  const startInterview = async (agentId: string) => {
    try {
      setStartingAgentId(agentId)
      await apiService.createSession(agentId)
      navigate('/interview')
    } catch (err) {
      console.error('Failed to start interview:', err)
      setError('Failed to start interview')
    } finally {
      setStartingAgentId(null)
    }
  }


  const agentColumns = [
    {
      key: 'name' as keyof Agent,
      label: 'Name',
      render: (_value: any, agent: Agent) => (
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-primary text-primary-foreground rounded-full flex items-center justify-center font-bold text-sm">
            {agent.name.charAt(0)}
          </div>
          <div>
            <div className="font-medium">{agent.name}</div>
            <div className="text-sm text-muted-foreground">{agent.industry || 'No industry'}</div>
          </div>
        </div>
      )
    },
    {
      key: 'description' as keyof Agent,
      label: 'Description',
      render: (_value: any) => (
        <div className="max-w-xs truncate">{_value || 'No description'}</div>
      )
    },
    {
      key: 'level' as keyof Agent,
      label: 'Level',
      render: (_value: any) => (
        <span className={`px-2 py-1 rounded text-xs ${
          _value === 'Junior' ? 'bg-lime-3 text-lime-11' :
          _value === 'Mid' ? 'bg-orange-3 text-orange-11' :
          _value === 'Senior' ? 'bg-orange-6 text-orange-11' :
          'bg-muted text-muted-foreground'
        }`}>
          {_value || 'Not specified'}
        </span>
      )
    },
    {
      key: 'created_at' as keyof Agent,
      label: 'Created',
      render: (_value: any) => new Date(_value).toLocaleDateString(),
      sortable: true
    },
    {
      key: 'id' as keyof Agent,
      label: 'Actions',
      render: (_value: any, agent: Agent) => {
        const isDefaultAgent = !agent.user_id // Default agents have no user_id
        return (
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              onClick={() => startInterview(agent.id)}
              disabled={startingAgentId === agent.id}
            >
              {startingAgentId === agent.id ? 'Starting…' : 'Start Interview'}
            </Button>
            {!isDefaultAgent && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setEditingAgent(agent)
                  setShowEditModal(true)
                }}
              >
                Edit
              </Button>
            )}
            {!isDefaultAgent && (
              <Button
                size="sm"
                variant="destructive"
                onClick={() => handleDeleteAgent(agent)}
              >
                Delete
              </Button>
            )}
            {isDefaultAgent && (
              <span className="text-sm text-muted-foreground px-2 py-1">
                Default Agent
              </span>
            )}
          </div>
        )
      }
    }
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">Loading agents...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="text-destructive text-lg mb-4">{error}</div>
          <Button onClick={loadAgents}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold">AI Agents</h1>
        <Button onClick={() => setShowAddModal(true)}>
          Add New Agent
        </Button>
      </div>

      <SearchableTable
        data={agents}
        columns={agentColumns}
        searchFields={['name', 'description', 'industry']}
        searchPlaceholder="Search agents by name, description, or industry..."
        emptyMessage="No agents found. Create your first agent above!"
      />

      {/* Add Agent Modal */}
      <AgentModal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        onSubmit={handleAddAgent}
        title="Add New Agent"
      />

      {/* Edit Agent Modal */}
      <AgentModal
        isOpen={showEditModal}
        onClose={() => {
          setShowEditModal(false)
          setEditingAgent(null)
        }}
        onSubmit={handleEditAgent}
        title="Edit Agent"
        agent={editingAgent}
      />

      {/* Delete Confirmation Modal */}
      <Modal 
        isOpen={showDeleteModal} 
        onClose={() => {
          setShowDeleteModal(false)
          setDeletingAgent(null)
        }} 
        title="Delete Agent" 
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            Are you sure you want to delete <strong>{deletingAgent?.name}</strong>? 
            This action cannot be undone.
          </p>
          
          <div className="flex space-x-3 pt-4">
            <Button 
              onClick={confirmDeleteAgent}
              variant="destructive"
              className="flex-1"
            >
              Delete Agent
            </Button>
            <Button 
              onClick={() => {
                setShowDeleteModal(false)
                setDeletingAgent(null)
              }}
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

// Agent Modal Component
interface AgentModalProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (agentData: Partial<Agent>) => void
  title: string
  agent?: Agent | null
}

function AgentModal({ isOpen, onClose, onSubmit, title, agent }: AgentModalProps) {
  const [formData, setFormData] = useState({
    name: agent?.name || '',
    description: agent?.description || '',
    industry: agent?.industry || '',
    level: agent?.level || ''
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(formData)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: value
    }))
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size="md">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="name" className="block text-sm font-medium mb-1">
            Agent Name
          </label>
          <input
            id="name"
            name="name"
            type="text"
            required
            value={formData.name}
            onChange={handleInputChange}
            className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
            placeholder="Enter agent name"
          />
        </div>

        <div>
          <label htmlFor="description" className="block text-sm font-medium mb-1">
            Description
          </label>
          <textarea
            id="description"
            name="description"
            value={formData.description}
            onChange={handleInputChange}
            rows={3}
            className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
            placeholder="Enter agent description"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="industry" className="block text-sm font-medium mb-1">
              Industry
            </label>
            <input
              id="industry"
              name="industry"
              type="text"
              value={formData.industry}
              onChange={handleInputChange}
              className="w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
              placeholder="e.g., Technology, Finance"
            />
          </div>

          <div>
            <label htmlFor="level" className="block text-sm font-medium mb-1">
              Level
            </label>
            <Select
              value={formData.level}
              onValueChange={(value: string) => setFormData(prev => ({ ...prev, level: value }))}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select level" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Junior">Junior</SelectItem>
                <SelectItem value="Mid">Mid</SelectItem>
                <SelectItem value="Senior">Senior</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex space-x-3 pt-4">
          <Button type="submit" className="flex-1">
            {agent ? 'Update Agent' : 'Create Agent'}
          </Button>
          <Button type="button" onClick={onClose} variant="outline" className="flex-1">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  )
}

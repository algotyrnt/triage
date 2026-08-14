import React, { useState, useEffect } from 'react';
import { ScreenId } from '@/types';
import { GithubIcon as Github } from '@/components/GithubIcon';
import { engineClient } from '@/services/engineClient';
import {
  Settings,
  CheckCircle2,
  ArrowRight,
  Shield,
  AlertTriangle,
  ExternalLink,
  RefreshCw,
  Loader2,
  Server,
  Key,
  Brain,
} from 'lucide-react';

interface SetupWizardPageProps {
  onNavigate: (screen: ScreenId) => void;
}

export const SetupWizardPage: React.FC<SetupWizardPageProps> = ({ onNavigate }) => {
  const [currentStep, setCurrentStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Step 1: Create App
  const [appCreated, setAppCreated] = useState(false);
  
  // Step 2: Install App
  const [appInstalled, setAppInstalled] = useState(false);
  const [repos, setRepos] = useState<{ owner: string; repo: string }[]>([]);
  
  // Step 3: OAuth
  const [oauthConfigured, setOauthConfigured] = useState(false);
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');

  // Step 4: AI Configuration
  const [llmConfigured, setLlmConfigured] = useState(false);
  const [geminiApiKey, setGeminiApiKey] = useState('');
  const [geminiModel, setGeminiModel] = useState('');
  
  // Step 5: Test
  const [connectionSuccess, setConnectionSuccess] = useState(false);
  const [appName, setAppName] = useState('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const step = params.get('setup_step');
    const isAppCreated = params.get('app_created');
    const isAppInstalled = params.get('app_installed');
    
    if (step) {
      setCurrentStep(parseInt(step, 10));
    }
    
    if (isAppCreated === 'true') {
      setAppCreated(true);
      if (!step) setCurrentStep(2);
    }
    
    if (isAppInstalled === 'true') {
      setAppInstalled(true);
      fetchRepos();
      if (!step) setCurrentStep(3);
    }

    checkStatus();
  }, []);

  const checkStatus = async () => {
    try {
      const status = await engineClient.getSetupStatus();
      if (status.github_app) setAppCreated(true);
      if (status.installation) {
        setAppInstalled(true);
        fetchRepos();
      }
      if (status.oauth) setOauthConfigured(true);
      if (status.llm) setLlmConfigured(true);
      
      if (status.github_app && status.installation && status.oauth && status.llm) {
        setCurrentStep(5);
      } else if (status.github_app && status.installation && status.oauth) {
        setCurrentStep(4);
      } else if (status.github_app && status.installation) {
        setCurrentStep(3);
      } else if (status.github_app) {
        setCurrentStep(2);
      }
    } catch (err) {
      console.error("Failed to check status", err);
    }
  };

  const fetchRepos = async () => {
    try {
      const installedRepos = await engineClient.getInstalledRepos();
      setRepos(installedRepos);
    } catch (err) {
      console.error("Failed to fetch repos", err);
    }
  };

  const handleCreateApp = async () => {
    setLoading(true);
    setError(null);
    try {
      const { manifest } = await engineClient.getSetupManifest(window.location.origin);
      
      // Create hidden form to POST manifest to GitHub
      const form = document.createElement('form');
      form.method = 'POST';
      form.action = 'https://github.com/settings/apps/new';
      
      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = 'manifest';
      input.value = JSON.stringify(manifest);
      
      form.appendChild(input);
      document.body.appendChild(form);
      form.submit();
    } catch (err: any) {
      setError(err.message || 'Failed to generate manifest');
      setLoading(false);
    }
  };

  const handleInstallApp = async () => {
    setLoading(true);
    setError(null);
    try {
      const { url } = await engineClient.getInstallUrl();
      window.location.href = url;
    } catch (err: any) {
      setError(err.message || 'Failed to get install URL');
      setLoading(false);
    }
  };

  const handleSaveOAuth = async () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      setError('Client ID and Secret are required');
      return;
    }
    
    setLoading(true);
    setError(null);
    try {
      await engineClient.saveOAuthConfig(clientId, clientSecret);
      setOauthConfigured(true);
      setCurrentStep(4);
    } catch (err: any) {
      setError(err.message || 'Failed to save OAuth config');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveLlmConfig = async () => {
    if (!geminiApiKey.trim() || !geminiModel.trim()) {
      setError('API Key and Model Name are required');
      return;
    }
    
    setLoading(true);
    setError(null);
    try {
      const success = await engineClient.saveLlmSetupConfig(geminiApiKey, geminiModel);
      if (success) {
        setLlmConfigured(true);
        setCurrentStep(5);
      } else {
        setError('Failed to save AI configuration');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to save AI configuration');
    } finally {
      setLoading(false);
    }
  };

  const handleTestConnection = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await engineClient.testSetupConnection();
      if (res.success) {
        setConnectionSuccess(true);
        if (res.app_name) setAppName(res.app_name);
      } else {
        setError(res.error || 'Connection test failed');
      }
    } catch (err: any) {
      setError(err.message || 'Connection test failed');
    } finally {
      setLoading(false);
    }
  };

  const handleComplete = () => {
    // Clear URL params
    window.history.replaceState({}, document.title, window.location.pathname);
    onNavigate('login');
  };

  const renderStepIndicator = () => (
    <div className="flex items-center justify-between mb-8 w-full max-w-2xl">
      {[1, 2, 3, 4, 5].map((step) => {
        const isCompleted = currentStep > step || 
          (step === 1 && appCreated) || 
          (step === 2 && appInstalled) || 
          (step === 3 && oauthConfigured) ||
          (step === 4 && llmConfigured);
          
        return (
          <React.Fragment key={step}>
            <div className="flex flex-col items-center">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center font-mono text-xs font-bold ${
                  currentStep === step
                    ? 'bg-black text-white'
                    : isCompleted
                    ? 'bg-emerald-600 text-white'
                    : 'bg-slate-200 text-slate-500'
                }`}
              >
                {isCompleted ? (
                  <CheckCircle2 className="w-4 h-4" />
                ) : (
                  step
                )}
              </div>
              <span className="text-[10px] uppercase tracking-wider font-mono text-slate-500 mt-2">
                {step === 1 ? 'Create App' : step === 2 ? 'Install' : step === 3 ? 'OAuth' : step === 4 ? 'AI Config' : 'Verify'}
              </span>
            </div>
            {step < 5 && (
              <div
                className={`flex-1 h-px mx-4 ${
                  isCompleted ? 'bg-emerald-600' : 'bg-slate-200'
                }`}
              />
            )}
          </React.Fragment>
        );
      })}
    </div>
  );

  return (
    <div className="min-h-[calc(100vh-100px)] bg-slate-50 flex flex-col items-center justify-center p-4">
      {renderStepIndicator()}
      
      <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-sm p-8 shadow-none space-y-8">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-sm flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5" />
            <div className="text-sm font-mono">{error}</div>
          </div>
        )}

        {/* Step 1: Create GitHub App */}
        {currentStep === 1 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Github className="w-6 h-6" /> Create GitHub App
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Triage requires a GitHub App to access repositories, read ASTs, and create issues for detected crashes. We'll automatically generate the configuration manifest for you.
              </p>
            </div>

            <div className="bg-slate-50 border border-slate-200 rounded-sm p-4 text-xs font-mono text-slate-700 space-y-2">
              <div className="flex items-center gap-2 font-semibold text-slate-900">
                <Shield className="w-4 h-4" /> App Permissions
              </div>
              <ul className="list-disc pl-5 space-y-1">
                <li>Read access to code (for AST isolation)</li>
                <li>Read/Write access to issues (for bug reports)</li>
                <li>Read access to metadata</li>
              </ul>
            </div>

            {appCreated ? (
              <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center justify-between">
                <div className="flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  GitHub App created successfully
                </div>
                <button
                  onClick={() => setCurrentStep(2)}
                  className="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-sm text-xs font-mono font-semibold transition-colors flex items-center gap-2"
                >
                  Next Step <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={handleCreateApp}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
              >
                {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <ExternalLink className="w-4 h-4" />}
                {loading ? 'Generating Manifest...' : 'Create GitHub App on GitHub'}
              </button>
            )}
          </div>
        )}

        {/* Step 2: Install App */}
        {currentStep === 2 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Server className="w-6 h-6" /> Install App into Organization
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Now that the App is created, it needs to be installed into your GitHub organization or personal account to grant access to specific repositories.
              </p>
            </div>

            {appInstalled ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  App installed successfully
                </div>
                
                {repos.length > 0 && (
                  <div className="bg-slate-50 border border-slate-200 rounded-sm p-4">
                    <h3 className="text-xs font-mono font-semibold text-slate-900 mb-2 uppercase tracking-wider">Granted Repositories</h3>
                    <ul className="space-y-1">
                      {repos.map((repo, idx) => (
                        <li key={idx} className="text-sm font-mono text-slate-700 flex items-center gap-2">
                          <Github className="w-3.5 h-3.5" />
                          {repo.owner}/{repo.repo}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                
                <button
                  onClick={() => setCurrentStep(3)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Continue to OAuth Setup <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={handleInstallApp}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
              >
                {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <ExternalLink className="w-4 h-4" />}
                {loading ? 'Preparing Install...' : 'Install GitHub App'}
              </button>
            )}
          </div>
        )}

        {/* Step 3: OAuth */}
        {currentStep === 3 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Key className="w-6 h-6" /> Dashboard Login Configuration
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                To allow users to log in to the Triage dashboard using GitHub, we need the OAuth credentials from your GitHub App.
              </p>
            </div>

            {oauthConfigured ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  OAuth credentials auto-configured from GitHub App
                </div>
                <button
                  onClick={() => setCurrentStep(4)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Continue to AI Configuration <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">Client ID</label>
                  <input
                    type="text"
                    value={clientId}
                    onChange={(e) => setClientId(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="Iv1.xxxxxxxxxxxx"
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">Client Secret</label>
                  <input
                    type="password"
                    value={clientSecret}
                    onChange={(e) => setClientSecret(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                  />
                </div>
                <button
                  onClick={handleSaveOAuth}
                  disabled={loading}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
                  Save OAuth Configuration
                </button>
              </div>
            )}
          </div>
        )}

        {/* Step 4: AI Configuration */}
        {currentStep === 4 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Brain className="w-6 h-6" /> AI Configuration
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Triage uses Gemini to automatically analyze crashes. Please provide your API credentials to continue.
              </p>
            </div>

            {llmConfigured ? (
              <div className="space-y-4">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-4 rounded-sm flex items-center gap-2 font-mono text-sm">
                  <CheckCircle2 className="w-5 h-5" />
                  AI configuration saved
                </div>
                <button
                  onClick={() => setCurrentStep(5)}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Continue to Verification <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">Gemini API Key</label>
                  <input
                    type="password"
                    value={geminiApiKey}
                    onChange={(e) => setGeminiApiKey(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="AIzaSy..."
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-mono font-semibold text-slate-700 uppercase tracking-wider">Model Name</label>
                  <input
                    type="text"
                    value={geminiModel}
                    onChange={(e) => setGeminiModel(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-sm px-3 py-2 text-sm font-mono focus:outline-none focus:border-black"
                    placeholder="e.g. gemini-1.5-flash"
                  />
                </div>
                <button
                  onClick={handleSaveLlmConfig}
                  disabled={loading}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
                  Save AI Configuration
                </button>
              </div>
            )}
          </div>
        )}

        {/* Step 5: Test Connection */}
        {currentStep === 5 && (
          <div className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
                <Settings className="w-6 h-6" /> Verify Setup
              </h2>
              <p className="text-sm text-slate-600 font-sans">
                Run a final test to ensure the dashboard can communicate with the Triage engine and the GitHub API successfully.
              </p>
            </div>

            {connectionSuccess ? (
              <div className="space-y-6">
                <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 p-6 rounded-sm flex flex-col items-center justify-center gap-4 text-center">
                  <div className="w-12 h-12 bg-emerald-100 rounded-full flex items-center justify-center">
                    <CheckCircle2 className="w-6 h-6 text-emerald-600" />
                  </div>
                  <div>
                    <h3 className="font-bold font-sans text-lg text-emerald-900">Setup Complete!</h3>
                    <p className="font-mono text-sm mt-1">GitHub App verified{appName ? `: ${appName}` : ''}</p>
                  </div>
                </div>
                <button
                  onClick={handleComplete}
                  className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
                >
                  Complete Setup & Continue to Login <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={handleTestConnection}
                disabled={loading}
                className="bg-black hover:bg-slate-800 text-white px-6 py-3 rounded-sm text-sm font-mono font-semibold transition-colors flex items-center justify-center gap-2 w-full"
              >
                {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
                {loading ? 'Testing...' : 'Test Connection'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

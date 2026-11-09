import { createRoot } from 'react-dom/client';
import './app/fonts.css';
import './app/globals.css';
import { App } from './App';
import { ApiProvider } from '@ff-resources/ui/api/ApiProvider';
import { setCurrentApi } from '@ff-resources/ui/api/registry';
import { ff13Api } from './api';
import { Providers } from './components/providers';

// Registro para os módulos fora do React (stores/loaders).
setCurrentApi(ff13Api);

const root = document.getElementById('root');
if (!root) {
  throw new Error('elemento #root não encontrado');
}

createRoot(root).render(
  <Providers>
    <ApiProvider api={ff13Api}>
      <App />
    </ApiProvider>
  </Providers>,
);

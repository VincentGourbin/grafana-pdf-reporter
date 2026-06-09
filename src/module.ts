// Entry point du frontend. Enregistre le link "Export PDF" dans le menu
// d'action des dashboards via l'extension point Grafana.
import { AppPlugin } from '@grafana/data';

export const plugin = new AppPlugin<{}>()
  .addLink({
    title: 'Export PDF',
    description: 'Generate a PDF report of this dashboard with cover page',
    targets: ['grafana/dashboard/action'],
    configure: (ctx?: { dashboard?: { uid?: string } }) => {
      const uid = ctx?.dashboard?.uid;
      if (!uid) {
        return undefined; // pas dans un dashboard → ne pas afficher
      }
      return {
        title: 'Export PDF',
        onClick: () => {
          // Reprendre from/to/theme depuis l'URL courante quand dispo.
          const params = new URLSearchParams(window.location.search);
          const q = new URLSearchParams();
          q.set('dashboard', uid);
          const from = params.get('from');
          const to = params.get('to');
          const theme = params.get('theme') ?? 'dark';
          if (from) q.set('from', from);
          if (to) q.set('to', to);
          q.set('theme', theme);
          const url = `/api/plugins/vincentgourbin-pdfreporter-app/resources/generate?${q.toString()}`;
          // Ouvre dans un nouvel onglet → le navigateur télécharge le PDF.
          window.open(url, '_blank', 'noopener');
        },
      };
    },
  });

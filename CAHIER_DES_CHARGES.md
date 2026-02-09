# Cahier des Charges - OKD TUI

## 1. Vision et Objectifs

### 1.1 Contexte
La console web OKD/OpenShift souffre de problèmes de performance récurrents :
- Temps de chargement longs (5-15s par page)
- Latence sur les actions (scale, restart, logs)
- Interface lourde pour des opérations quotidiennes simples
- Timeouts fréquents sur les clusters chargés

### 1.2 Objectif
Créer un outil TUI (Terminal User Interface) qui remplace 80% des interactions quotidiennes avec la console web OKD, en offrant une expérience **instantanée**, **sécurisée** et **intuitive** directement dans le terminal.

### 1.3 Proposition de valeur
| Critère | Console Web OKD | OKD TUI (cible) |
|---------|----------------|-----------------|
| Temps de réponse | 3-15s | < 200ms |
| Navigation entre ressources | 5+ clics | 1-2 touches |
| Utilisation mémoire | ~500MB (navigateur) | < 50MB |
| Accès aux logs | 3 clics + scroll lent | 1 touche, streaming |

### 1.4 Public cible
- **Développeurs** : consultation des pods, logs, déploiements quotidiens
- **DevOps / SRE** : troubleshooting, scaling, gestion multi-namespace
- **Ops / Admins** : vue d'ensemble cluster, gestion des quotas et limites

### 1.5 Dépendances externes
| Dépendance | Obligatoire | Détail |
|-----------|-------------|--------|
| `~/.kube/config` ou `$KUBECONFIG` | Oui | Fichier kubeconfig valide avec token/cert |
| `oc` CLI | **Non** | L'outil est 100% standalone via client-go. `oc` n'est pas requis |
| Accès réseau au cluster | Oui | API server accessible (port 6443 typiquement) |
| metrics-server | Non | Requis uniquement pour les métriques CPU/Mem. Dégradation gracieuse sinon |
| Terminal 256 couleurs | Non | Fallback automatique en 16 couleurs / ASCII si terminal basique |

---

## 2. Parcours Utilisateurs

Les parcours définissent le comportement attendu de l'outil dans les scénarios réels. Chaque parcours est la référence pour l'implémentation.

### PU-01 : Premier lancement

```
Utilisateur lance `okd-tui` pour la première fois.

1. L'outil cherche le kubeconfig :
   a. $KUBECONFIG si défini
   b. Sinon ~/.kube/config
   c. Sinon → ERREUR : écran "Aucun kubeconfig trouvé"
      Affiche : "Configurez votre accès avec : oc login <cluster-url>"
      Touche q pour quitter

2. Si kubeconfig trouvé, tente la connexion au contexte courant :
   a. Succès → affiche la vue Pods du namespace courant
   b. Token expiré (401) → ERREUR : écran "Token expiré"
      Affiche : "Reconnectez-vous avec : oc login <cluster-url>"
      L'URL du cluster est extraite du kubeconfig et affichée
   c. Cluster injoignable (timeout) → ERREUR : écran "Cluster injoignable"
      Affiche l'URL du serveur API et le message d'erreur réseau
      Propose 'r' pour réessayer

3. Aucun fichier de config okd-tui n'est nécessaire au premier lancement.
   Tout fonctionne avec les défauts.
```

### PU-02 : "Mon pod crashloop, je veux comprendre pourquoi"

C'est le cas d'usage #1, le plus fréquent.

```
1. L'outil s'ouvre sur la vue Pods du namespace courant
2. L'utilisateur voit immédiatement les pods en CrashLoopBackOff
   (colorés en rouge, triés par statut si le tri est actif)
3. Il navigue avec j/k jusqu'au pod en erreur
4. Appuie sur Enter → bascule en vue Logs
   - Les 200 dernières lignes s'affichent instantanément
   - Les logs scrollent avec pgup/pgdn
5. Si le container a redémarré et les logs actuels sont vides :
   - Appuie sur 'p' → affiche les logs du container précédent (--previous)
6. Il veut voir les events du pod :
   - Appuie sur Esc pour revenir à la liste
   - Appuie sur 'y' → affiche le YAML du pod
   - La section .status.conditions et les events sont visibles
7. Il décide de relancer le pod :
   - Appuie sur Esc pour revenir à la liste
   - Appuie sur 'd' (delete)
   - Dialog de confirmation s'affiche : "Supprimer <pod-name> ? [y/N]"
   - Appuie sur 'y' → pod supprimé
   - La liste se rafraîchit automatiquement, le nouveau pod apparaît
```

### PU-03 : "Je veux scaler mon application"

```
1. L'utilisateur est sur la vue Pods, appuie sur '2' pour aller
   sur la vue Deployments
2. Il voit la liste des deployments avec les colonnes :
   NAME | READY | AVAILABLE | AGE | IMAGE
3. Il navigue jusqu'au deployment cible
4. Appuie sur '+' → le replica count passe de 2 à 3
   - Feedback immédiat : la colonne READY affiche "2/3" (en jaune, scaling en cours)
   - Quelques secondes plus tard, passe à "3/3" (en vert)
5. S'il veut scaler à un nombre précis :
   - Appuie sur 's' → input numérique "Replicas: _"
   - Tape le nombre, Enter pour valider
   - Si le nombre est > 10, un warning s'affiche : "Scale à 15 replicas ?"
```

### PU-04 : "Je veux changer de projet/namespace"

```
1. Depuis n'importe quelle vue, l'utilisateur appuie sur '1'
   → bascule sur la vue Projects
2. La liste des namespaces accessibles s'affiche (triée alphabétiquement)
3. Il tape '/' pour activer le filtre
4. Tape "api" → seuls les projets contenant "api" restent affichés
5. Navigue avec j/k, appuie sur Enter
6. Le namespace actif change (visible dans la barre de contexte en haut)
7. L'outil bascule automatiquement sur la vue Pods du nouveau namespace
```

### PU-05 : "Je veux ouvrir un shell dans un pod"

```
1. Depuis la vue Pods, l'utilisateur navigue jusqu'au pod cible
2. Appuie sur 's' (shell)
3. Si le pod a plusieurs containers :
   - Un sélecteur s'affiche : "Container: [app] [sidecar] [init]"
   - Il choisit le container
4. La TUI se met en pause (programme alternatif du terminal)
5. Un exec interactif s'ouvre (comme 'oc exec -it <pod> -- /bin/sh')
6. L'utilisateur travaille dans le shell
7. À la sortie (exit ou Ctrl+D), la TUI reprend exactement où elle était
   (même vue, même curseur, même filtre)

TECHNIQUE : On utilise os/exec avec le binaire 'oc' si disponible,
sinon on utilise client-go remotecommand avec SPDY.
Le terminal est restitué à Bubbletea via tea.ExecProcess().
```

### PU-06 : "Je veux vérifier l'état de mon déploiement après un merge"

```
1. L'utilisateur ouvre okd-tui, arrive sur les pods
2. Il appuie sur '2' → vue Deployments
3. Il voit que son deployment est en "Progressing"
   (colonne READY affiche "1/2" en jaune)
4. Les données se mettent à jour en temps réel (watch API)
5. Après 30s, READY passe à "2/2" en vert
6. Il appuie sur Enter pour voir le détail du deployment
7. Il vérifie l'image déployée pour confirmer que c'est le bon tag
```

### PU-07 : "J'ai une erreur, impossible de faire quoi que ce soit"

Parcours dégradés -- l'outil doit toujours rester utilisable.

```
SCÉNARIO A : Pas de droits sur le namespace
  1. L'utilisateur switch sur un namespace
  2. La requête list pods retourne 403 Forbidden
  3. Affichage : "Accès refusé au namespace 'kube-system'"
     "Vos droits ne permettent pas de lister les pods dans ce namespace."
     "Essayez un autre namespace avec [1] Projects"
  4. La barre de contexte reste affichée, l'utilisateur peut naviguer

SCÉNARIO B : Cluster injoignable en cours d'utilisation
  1. L'utilisateur travaille normalement
  2. Le réseau coupe → les requêtes API timeout
  3. Affichage : bannière orange en haut "⚠ Connexion perdue - données en cache"
     Les dernières données restent affichées (pas d'écran vide)
     Retry automatique toutes les 5s en background
  4. Quand la connexion revient :
     Bannière verte "✓ Reconnecté" pendant 3s, puis disparaît
     Les données se rafraîchissent automatiquement

SCÉNARIO C : Token expiré en cours d'utilisation
  1. L'utilisateur fait une action (delete pod, scale)
  2. L'API retourne 401 Unauthorized
  3. Affichage : "Session expirée. Reconnectez-vous :"
     "  oc login https://api.my-cluster:6443"
     "Puis appuyez sur 'r' pour reconnecter"
  4. La TUI ne quitte PAS. L'utilisateur fait oc login dans un autre terminal,
     revient dans la TUI, appuie sur 'r', le kubeconfig est rechargé.

SCÉNARIO D : Action échoue (ex: delete un pod qui a déjà disparu)
  1. L'utilisateur tente de supprimer un pod
  2. L'API retourne 404 Not Found
  3. Toast message : "Pod 'xxx' déjà supprimé"
  4. La liste se rafraîchit, le pod disparaît
  5. Aucune perturbation, l'utilisateur continue
```

---

## 3. Exigences Fonctionnelles

### Périmètre MVP (Phase 1 -- le strict minimum utile)

Seulement ces features. Tout le reste est hors scope du MVP.

| ID | Feature | Critère de done |
|----|---------|-----------------|
| F-01 | Connexion kubeconfig | Se connecte au contexte courant. Affiche erreur claire si impossible |
| F-02 | Vue Pods | Liste les pods : nom, statut (coloré), ready, restarts, age |
| F-03 | Logs pod | Enter sur un pod → affiche les 200 dernières lignes, scroll pgup/pgdn |
| F-04 | Logs previous | Touche 'p' en vue logs → logs du container précédent |
| F-05 | Delete pod | Touche 'd' → confirmation y/N → suppression |
| F-06 | Vue Deployments | Liste : nom, ready, available, age, image |
| F-07 | Scale +/- | Touches +/- sur un deployment → modifie replicas |
| F-08 | Vue Projects | Liste namespaces, Enter → switch + retour vue pods |
| F-09 | Filtre fuzzy | '/' → tape un mot → filtre la liste en temps réel |
| F-10 | Navigation vim | j/k, g/G, pgup/pgdn, esc pour retour |
| F-11 | Barre contexte | Toujours visible : cluster, namespace, nb items |
| F-12 | Barre aide | Raccourcis affichés en bas, contextuels à la vue |
| F-13 | Gestion erreurs | 401, 403, timeout, 404 → messages clairs (cf. PU-07) |
| F-14 | Refresh | Touche 'r' → recharge les données |

**Ce qui n'est PAS dans le MVP** : exec/shell, watch temps réel, services, routes, configmaps, secrets, events, YAML, multi-cluster, builds, métriques, audit, config file.

### Phase 2 (après validation du MVP)

| ID | Feature | Dépendance |
|----|---------|------------|
| F-20 | Watch temps réel | Remplace le polling par des websockets K8s |
| F-21 | Exec/Shell dans pod | Requiert PTY forwarding (cf. section 6.3) |
| F-22 | Vue Events | Stream events du namespace |
| F-23 | Vue YAML | Affichage YAML formaté d'une ressource |
| F-24 | Logs multi-container | Sélecteur de container si pod multi-container |
| F-25 | Tri des colonnes | Tab pour changer le tri (nom, age, status, restarts) |
| F-26 | Cache avec TTL | 5s pods, 30s namespaces, invalidation sur mutation |
| F-27 | Config file YAML | `~/.config/okd-tui/config.yaml` |

### Phase 3 (OKD-spécifique)

| ID | Feature | API Group |
|----|---------|-----------|
| F-30 | Routes OKD | `route.openshift.io/v1` |
| F-31 | DeploymentConfigs | `apps.openshift.io/v1` |
| F-32 | Builds / BuildConfigs | `build.openshift.io/v1` |
| F-33 | ImageStreams | `image.openshift.io/v1` |
| F-34 | Détection cluster type | Au démarrage : tester si les API groups OKD existent. Si non → mode K8s vanilla, les tabs OKD sont masquées |

### Phase 4 (polish)

| ID | Feature |
|----|---------|
| F-40 | Multi-cluster (switch context kubeconfig) |
| F-41 | Couleur par environnement (prod = rouge) |
| F-42 | Métriques CPU/Mem (metrics.k8s.io API) |
| F-43 | Audit trail local |
| F-44 | Namespace read-only configurable |
| F-45 | Services et ConfigMaps/Secrets |

---

## 4. Exigences UX

### 4.1 Principes

1. **Vim-like** : j/k/g/G, /filter, esc pour retour. Pas de keybindings exotiques.
2. **Feedback immédiat** : spinner pendant le chargement, message de succès/erreur après chaque action.
3. **Contexte toujours visible** : on ne doit jamais se demander "je suis sur quel cluster ? quel namespace ?"
4. **Prévention des erreurs** : confirmation pour toute action destructive. Renforcée pour les namespaces prod.
5. **Progressive disclosure** : l'écran par défaut est simple. Les détails sont à un Enter de distance.

### 4.2 Keybindings complets

```
NAVIGATION GLOBALE
──────────────────────────────────────────
  1           Vue Projects
  2           Vue Pods
  3           Vue Deployments
  Tab         Vue suivante
  /           Filtre fuzzy
  ?           Aide (overlay)
  r           Rafraîchir les données
  q, Ctrl+C   Quitter (ou retour si dans sous-vue)

NAVIGATION LISTE
──────────────────────────────────────────
  j, ↓        Descendre
  k, ↑        Monter
  g            Premier élément
  G            Dernier élément
  Ctrl+D       Page down
  Ctrl+U       Page up
  Enter        Action principale (détail / sélection)
  Esc          Retour à la vue précédente

ACTIONS SUR PODS (vue Pods)
──────────────────────────────────────────
  Enter       Logs du pod
  p           Logs précédents (--previous) [en vue logs]
  d           Delete pod (avec confirmation)
  s           Shell/Exec (Phase 2)
  y           YAML du pod (Phase 2)
  c           Copier le nom dans le clipboard

ACTIONS SUR DEPLOYMENTS (vue Deployments)
──────────────────────────────────────────
  +           Scale up (+1 replica)
  -           Scale down (-1 replica)
  s           Scale à un nombre précis (input)

FILTRE
──────────────────────────────────────────
  /           Activer le filtre
  [texte]     Filtre en temps réel
  Enter       Valider et garder le filtre
  Esc         Annuler le filtre
```

### 4.3 Layout

```
┌──────────────────────────────────────────────────────────────┐
│  OKD TUI   ctx:my-cluster   ns:my-project                   │  <- Barre contexte (toujours visible)
├──────────────────────────────────────────────────────────────┤
│  [1] Projects  [2] Pods  [3] Deployments                     │  <- Tabs (vue active en surbrillance)
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  NAME                        STATUS      READY  RESTARTS AGE │  <- Header colonnes (gris, underline)
│  ───────────────────────────────────────────────────────────  │
│ >api-server-7d4f8b-x2k      Running     1/1    0        2d  │  <- Ligne sélectionnée (fond gris)
│  api-server-7d4f8b-9mn      Running     1/1    0        2d  │
│  worker-5c8d9f-abc          Running     3/3    2        5h  │
│  redis-master-0             Running     1/1    0        15d │
│  celery-beat-6f7g8h-ij      CrashLoop   0/1    47       1d  │  <- Statut coloré (rouge)
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ PODS | my-project | 23 items   j/k:nav enter:logs d:del q:q │  <- Barre statut + aide contextuelle
└──────────────────────────────────────────────────────────────┘
```

**Layout responsive :**

| Largeur terminal | Comportement |
|-----------------|-------------|
| < 60 colonnes | Colonnes tronquées : NAME + STATUS uniquement. Autres colonnes masquées |
| 60-100 colonnes | Colonnes principales : NAME, STATUS, READY, AGE |
| 100-150 colonnes | Toutes les colonnes standards |
| > 150 colonnes | Colonnes étendues : ajout NODE, IMAGE, NAMESPACE |
| Hauteur < 15 lignes | Barre de tabs masquée, compacte |

### 4.4 Confirmations

**Action standard (namespace non-prod) :**
```
Supprimer le pod api-server-7d4f8b-x2k ? [y/N] _
```
Simple y/N. Défaut N.

**Action sur namespace prod :**
```
┌────────────────────────────────────────────────┐
│  ⚠ NAMESPACE PRODUCTION                       │
│                                                │
│  Action : Supprimer pod                        │
│  Pod    : api-server-7d4f8b-x2k               │
│  NS     : production                           │
│                                                │
│  Tapez "api-server-7d4f8b-x2k" pour confirmer │
│  > _                                           │
│                                                │
│  [Esc] Annuler                                 │
└────────────────────────────────────────────────┘
```
Il faut taper le nom complet de la ressource. Pas de raccourci.

**Détection namespace prod :**
Le namespace est considéré comme prod si son nom contient un des patterns configurés. Par défaut : `prod`, `production`, `prd`, `live`.

### 4.5 Couleurs

| Élément | Couleur | Code |
|---------|---------|------|
| Branding OKD | Rouge | `#EE0000` |
| Contexte cluster | Bleu K8s | `#326CE5` |
| Namespace actif | Violet | `#7D56F4` |
| Running / Active | Vert | `#04B575` |
| Pending / Scaling | Jaune | `#FFBD2E` |
| Failed / CrashLoop | Rouge | `#FF6B6B` |
| Texte secondaire | Gris | `#626262` |
| Ligne sélectionnée | Fond gris | `#333333` |
| Bannière PROD | Fond rouge | `#8B0000` |
| Bannière connexion perdue | Fond orange | `#CC7700` |
| Toast succès | Vert | `#04B575` |
| Toast erreur | Rouge | `#FF6B6B` |

**Fallback 16 couleurs :** si le terminal ne supporte pas 256 couleurs (détection via `$TERM`), utiliser les couleurs ANSI de base : rouge, vert, jaune, bleu, magenta, cyan, blanc, gris.

---

## 5. Gestion des Erreurs (exhaustif)

Chaque erreur a un comportement défini. Le TUI ne doit **jamais** crasher ni afficher un stacktrace.

### 5.1 Erreurs de connexion

| Code | Situation | Affichage | Action utilisateur |
|------|-----------|-----------|-------------------|
| - | Pas de kubeconfig | "Aucun kubeconfig trouvé. Configurez avec : oc login \<url\>" | q pour quitter |
| - | Kubeconfig malformé | "Kubeconfig invalide : \<détail erreur\>" | q pour quitter |
| - | Pas de contexte courant | "Aucun contexte actif dans le kubeconfig" | q pour quitter |
| TCP timeout | Cluster injoignable | "Cluster injoignable : \<api-url\>\n\<erreur réseau\>" | 'r' pour réessayer |
| TLS error | Certificat invalide | "Certificat TLS invalide pour \<api-url\>. Vérifiez votre kubeconfig." | q pour quitter |
| 401 | Token expiré | "Session expirée. Reconnectez-vous :\n  oc login \<api-url\>\nPuis 'r' pour reconnecter" | 'r' après re-login |

### 5.2 Erreurs en cours d'utilisation

| Code | Situation | Affichage | Comportement |
|------|-----------|-----------|-------------|
| 403 | Pas de droits RBAC | Toast : "Accès refusé au namespace '\<ns\>'" | Reste sur l'écran courant. Données vides avec message explicatif |
| 404 | Ressource disparue | Toast : "Pod '\<name\>' introuvable (déjà supprimé ?)" | Rafraîchit la liste automatiquement |
| 409 | Conflit (scale pendant update) | Toast : "Conflit : la ressource a été modifiée. Réessayez." | L'utilisateur peut re-tenter |
| 422 | Paramètre invalide | Toast : "Valeur invalide : \<détail\>" | Reste sur l'écran de saisie |
| 429 | Rate limited | Toast : "Trop de requêtes. Pause 2s..." | Retry automatique après 2s |
| 500+ | Erreur serveur | Toast : "Erreur serveur (\<code\>). Réessayez avec 'r'." | Données en cache restent affichées |
| TCP timeout | Connexion perdue | Bannière : "⚠ Connexion perdue - données en cache" | Retry auto 5s. Bannière verte quand ça revient |

### 5.3 Erreurs spécifiques aux actions

| Action | Erreur possible | Comportement |
|--------|----------------|-------------|
| Delete pod | Pod en Terminating | Toast : "Pod en cours de suppression, patientez" |
| Scale deployment | Replicas < 0 | Bloqué côté client. Minimum = 0 |
| Scale deployment | Quota dépassé | Toast : "Quota replicas dépassé pour ce namespace" |
| Logs | Pod pas encore Running | Toast : "Le pod n'a pas encore démarré. Pas de logs disponibles" |
| Logs | Container en attente (Init) | Affiche les logs du init container avec mention |
| Exec/Shell | Pas de shell dans le container | Toast : "/bin/sh introuvable. Essayez un autre container" |
| Exec/Shell | Container pas ready | Toast : "Container pas prêt. Attendez qu'il soit Running" |

### 5.4 Règle générale

```
SI erreur récupérable (réseau, 401, 429, 500) :
    → Afficher le message, garder les données en cache, proposer 'r'
    → Ne JAMAIS vider l'écran

SI erreur fatale (kubeconfig absent, TLS) :
    → Écran d'erreur plein avec instruction claire
    → Touche q pour quitter proprement

SI erreur d'action (403, 404, 409, 422) :
    → Toast de 5s, pas de changement de vue
    → L'utilisateur peut continuer à naviguer

JAMAIS :
    → Panic / crash
    → Stacktrace Go visible
    → Écran vide sans explication
    → Message générique "une erreur est survenue"
```

---

## 6. Architecture Technique

### 6.1 Stack

| Composant | Technologie | Justification |
|-----------|-------------|---------------|
| Langage | **Go 1.22+** | Écosystème natif K8s, binaire unique, cross-compile facile |
| Framework TUI | **Bubbletea** (Charm) | ELM architecture, composable, excellente gestion terminal |
| Styling | **Lipgloss** (Charm) | Styling déclaratif pour terminal |
| Composants | **Bubbles** (Charm) | Table, textinput, viewport, spinner |
| Client K8s | **client-go v0.31+** | Client officiel, watch, exec, logs |
| Client OKD | **API REST directe** | Pour Route, DC, Build -- évite la dépendance openshift-client-go qui est lourde et parfois en retard de version |
| Config | **Viper** | Fichier YAML + env vars + défauts |

**Pourquoi pas openshift-client-go ?**
Le module `github.com/openshift/client-go` entraîne une chaîne de dépendances massive et des conflits de version fréquents avec client-go upstream. Pour les CRDs OKD (Routes, DC, Builds), on utilise le client REST dynamique de client-go (`k8s.io/client-go/dynamic`) avec les types définis localement. C'est plus léger et découplé.

### 6.2 Authentification et sécurité

**Flux d'authentification :**

```
Démarrage
    │
    ▼
Lire kubeconfig ($KUBECONFIG ou ~/.kube/config)
    │
    ▼
Extraire le contexte courant
    │
    ├─ Auth par token (le plus courant avec OKD)
    │   → Le token est dans le kubeconfig (posé par `oc login`)
    │   → client-go l'envoie automatiquement en header Authorization: Bearer
    │
    ├─ Auth par certificat client (clusters internes)
    │   → cert + key dans le kubeconfig
    │   → client-go gère le TLS mutuel
    │
    └─ Auth par exec plugin (OIDC, AWS IAM, etc.)
        → kubeconfig contient une commande exec
        → client-go exécute la commande pour obtenir un token frais
        → Supporte le refresh automatique

L'outil NE stocke AUCUN credential.
L'outil NE modifie JAMAIS le kubeconfig.
L'outil NE gère PAS le login -- c'est le rôle de `oc login`.
```

**Gestion du token expiré :**

```
Toute requête API
    │
    ▼
Erreur 401 ?
    │
    ├─ Non → ok, continuer
    │
    └─ Oui → Marquer le client comme "expiré"
              Afficher le message de reconnexion (PU-07 scénario C)
              Stopper les watch streams
              Attendre que l'utilisateur appuie sur 'r'
              Sur 'r' → recharger le kubeconfig depuis le disque
                         recréer le clientset
                         tester la connexion
                         si ok → reprendre
                         si 401 encore → re-afficher le message
```

### 6.3 Exec / Shell (Phase 2) -- Design technique

L'exec dans un pod est le point le plus complexe techniquement.

```
Approche 1 (préférée) : tea.ExecProcess() + oc exec
    Si `oc` est trouvé dans le PATH :
    1. Bubbletea met le TUI en pause via tea.ExecProcess()
    2. On lance : oc exec -it -n <ns> <pod> -c <container> -- /bin/sh
    3. oc gère le PTY, le resize, les signaux
    4. Quand l'utilisateur quitte (exit/Ctrl+D), Bubbletea reprend

    Avantage : simple, fiable, gère le SPDY/WebSocket transparent
    Inconvénient : nécessite `oc` dans le PATH

Approche 2 (fallback) : client-go remotecommand
    Si `oc` n'est pas disponible :
    1. Utiliser k8s.io/client-go/tools/remotecommand
    2. Créer un SPDYExecutor avec stdin/stdout/stderr
    3. Problème : la gestion du PTY raw mode + resize est manuelle
    4. Il faut :
       - Passer stdin en raw mode (x/term)
       - Écouter SIGWINCH pour le resize
       - Restaurer le terminal proprement en sortie (defer)

    Plus complexe, mais 100% standalone.

Décision : implémenter l'approche 1 en premier. Approche 2 en P3 si demandée.
```

### 6.4 APIs OKD utilisées

| Ressource | API Group | Version | Verbes utilisés | Phase |
|-----------|-----------|---------|----------------|-------|
| Pods | `core` | v1 | list, get, delete, log | 1 |
| Namespaces | `core` | v1 | list | 1 |
| Deployments | `apps` | v1 | list, get, update (scale) | 1 |
| ReplicaSets | `apps` | v1 | list | 2 |
| Events | `core` | v1 | list, watch | 2 |
| Services | `core` | v1 | list, get | 4 |
| ConfigMaps | `core` | v1 | list, get | 4 |
| Secrets | `core` | v1 | list, get | 4 |
| Routes | `route.openshift.io` | v1 | list, get | 3 |
| DeploymentConfigs | `apps.openshift.io` | v1 | list, get, update | 3 |
| BuildConfigs | `build.openshift.io` | v1 | list, get, instantiate | 3 |
| Builds | `build.openshift.io` | v1 | list, get, log | 3 |
| ImageStreams | `image.openshift.io` | v1 | list, get | 3 |
| PodMetrics | `metrics.k8s.io` | v1beta1 | list | 4 |

**Détection OKD vs Kubernetes vanilla :**
Au démarrage, on tente un `GET /apis/route.openshift.io` (API discovery). Si l'API group existe → mode OKD (tabs OKD activées). Sinon → mode Kubernetes vanilla (tabs OKD masquées, pas d'erreur).

### 6.5 RBAC minimal requis

L'outil a besoin au minimum de ces droits pour fonctionner :

```yaml
# RBAC minimum pour le MVP
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["list"]
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["list", "get", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments", "deployments/scale"]
    verbs: ["list", "get", "update"]
```

Si un verbe est refusé (403), l'outil fonctionne en mode dégradé :
- Pas de droit `delete` → le keybinding 'd' est désactivé (grisé dans l'aide)
- Pas de droit `list namespaces` → la vue Projects affiche uniquement le namespace courant
- Pas de droit `update scale` → les touches +/- sont désactivées

### 6.6 Structure du projet (MVP)

```
okd-tui/
├── cmd/
│   └── main.go                  # Point d'entrée, init client, lance le TUI
├── internal/
│   ├── k8s/
│   │   ├── client.go            # Init client-go, gestion kubeconfig, reconnexion
│   │   ├── pods.go              # List, Delete, GetLogs
│   │   ├── deployments.go       # List, Scale
│   │   └── namespaces.go        # List
│   ├── tui/
│   │   ├── app.go               # Modèle Bubbletea principal, routing entre vues
│   │   ├── styles.go            # Styles Lipgloss centralisés
│   │   ├── keys.go              # Keybindings (keymap)
│   │   ├── view_pods.go         # Vue liste pods
│   │   ├── view_deployments.go  # Vue liste deployments
│   │   ├── view_projects.go     # Vue liste projets
│   │   ├── view_logs.go         # Vue logs scrollable
│   │   ├── statusbar.go         # Composant barre de statut
│   │   ├── confirm.go           # Composant dialog confirmation
│   │   ├── toast.go             # Composant notifications
│   │   └── filter.go            # Composant filtre fuzzy
│   └── config/
│       └── config.go            # Defaults, detection prod namespace
├── go.mod
├── go.sum
├── Makefile
└── CAHIER_DES_CHARGES.md
```

~20 fichiers pour le MVP. Pas plus.

---

## 7. Sécurité

| ID | Exigence | Implémentation | Phase |
|----|----------|---------------|-------|
| S-01 | Aucun stockage de credentials | Lecture seule du kubeconfig via client-go | MVP |
| S-02 | Aucune modification du kubeconfig | Jamais d'écriture dans ~/.kube/ | MVP |
| S-03 | Secrets masqués par défaut | Afficher `*****` puis révéler sur action explicite | P4 |
| S-04 | Confirmation renforcée pour prod | Taper le nom complet de la ressource | MVP |
| S-05 | Pas de phone home | Zero requête réseau hors API cluster | MVP |
| S-06 | Pas d'escalade de privilèges | N'utilise que les droits du kubeconfig | MVP |
| S-07 | Actions destructives loguées | Fichier audit local (optionnel) | P4 |
| S-08 | Namespace read-only configurable | Config YAML, bloque delete/scale/exec | P4 |

---

## 8. Stratégie de Tests

### 8.1 Tests unitaires

| Couche | Ce qu'on teste | Comment |
|--------|---------------|---------|
| `internal/k8s/` | Parsing des réponses API, formatage des données | Structs K8s en entrée, vérifier les PodInfo/DeploymentInfo en sortie |
| `internal/tui/` | Logique de filtre, tri, pagination, troncature | Fonctions pures, pas de dépendance I/O |
| `internal/config/` | Parsing config, détection prod patterns | Fichiers YAML de test |

### 8.2 Tests d'intégration

| Test | Outil | Description |
|------|-------|-------------|
| API K8s | **envtest** (controller-runtime) | Lance un API server local en mémoire. On teste les List/Delete/Scale réels |
| Alternative | **fake clientset** (client-go/kubernetes/fake) | Mock du clientset. Plus léger, suffisant pour le MVP |
| E2E (optionnel) | **kind** (Kubernetes in Docker) | Cluster local complet. Pour les tests exec/logs/watch |

### 8.3 Tests TUI

| Aspect | Méthode |
|--------|---------|
| Rendu des vues | Bubbletea fournit `tea.TestModel()`. On envoie des messages, on vérifie la sortie `.View()` |
| Navigation | Simuler des KeyMsg, vérifier que le modèle change de vue/curseur |
| Confirmation | Simuler la séquence 'd' → 'y', vérifier que DeletePod est appelé |

### 8.4 Ce qu'on ne teste PAS

- Le rendu pixel-perfect dans chaque terminal (trop de variantes)
- La performance réseau réelle (dépend du cluster)
- L'authentification OKD (on fait confiance à client-go)

---

## 9. Configuration

Emplacement : `~/.config/okd-tui/config.yaml`

Aucun fichier de config n'est requis. Tout a un défaut sensible.

```yaml
# Détection namespaces de production
# Actions destructives requièrent la saisie du nom complet
prod_patterns:
  - prod
  - production
  - prd
  - live

# Namespaces en lecture seule (glob patterns)
# Aucune action destructive autorisée
readonly_namespaces: []
  # - kube-system
  # - openshift-*

# Cache TTL (Phase 2)
# cache:
#   pods: 5s
#   namespaces: 30s
#   deployments: 10s

# Audit (Phase 4)
# audit:
#   enabled: false
#   file: ~/.config/okd-tui/audit.log
```

---

## 10. Plan de livraison

### Phase 1 -- MVP

**Objectif :** un outil utilisable au quotidien pour les 3 opérations les plus fréquentes : voir les pods, lire les logs, scaler les deployments.

| Tâche | Dépend de | Critère de done |
|-------|-----------|-----------------|
| T-01 : Scaffolding projet Go + Bubbletea | - | `go build` compile, le binaire lance un écran vide |
| T-02 : Client K8s (connexion kubeconfig) | T-01 | Se connecte au cluster, affiche le contexte. Erreurs gérées (PU-01) |
| T-03 : Vue Pods (liste) | T-02 | Affiche les pods du namespace courant. Colonnes : name, status (coloré), ready, restarts, age |
| T-04 : Navigation vim (j/k/g/G/pgup/pgdn) | T-03 | Cursor se déplace, scroll quand la liste dépasse l'écran |
| T-05 : Vue Logs | T-03 | Enter sur un pod → logs 200 lignes. Scroll pgup/pgdn. 'p' → previous. Esc → retour |
| T-06 : Delete pod | T-03 | 'd' → confirmation → delete. Refresh auto après |
| T-07 : Vue Deployments | T-02 | Liste deployments. Colonnes : name, ready, available, age, image |
| T-08 : Scale +/- | T-07 | +/- modifient les replicas. Feedback visuel immédiat |
| T-09 : Vue Projects | T-02 | Liste namespaces. Enter → switch + retour pods |
| T-10 : Filtre fuzzy | T-03 | '/' → filtre en temps réel sur le nom |
| T-11 : Barre contexte + statut + aide | T-03 | Toujours visible. Cluster, namespace, nb items, keybindings contextuels |
| T-12 : Gestion erreurs complète | T-02 | 401, 403, timeout, 404 → messages clairs, jamais de crash |
| T-13 : Confirmation renforcée prod | T-06 | Namespace prod → taper le nom complet |

**Ordre d'implémentation recommandé :**
T-01 → T-02 → T-03 → T-04 → T-11 → T-10 → T-05 → T-06 → T-13 → T-07 → T-08 → T-09 → T-12

**Done quand :** un développeur peut lancer l'outil, naviguer entre 3 vues, lire des logs, supprimer un pod, scaler un deployment, changer de namespace. Tout ça sans crash, avec des messages d'erreur clairs.

---

## 11. Risques et mitigation

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| Token expiré sans prévenir | Élevé | Moyen | Détection 401, message de re-login, touche 'r' pour reconnecter |
| Cluster lent (5000+ pods) | Moyen | Élevé | Pagination serveur (limit=500), lazy loading, pas de list --all-namespaces |
| openshift-client-go incompatible | Moyen | Moyen | Pas de dépendance dessus. Client REST dynamique pour les CRDs OKD |
| Rendu cassé sur terminal exotique | Faible | Faible | Bubbletea gère la majorité des terminaux. Fallback 16 couleurs |
| RBAC trop restrictif | Moyen | Moyen | Mode dégradé : désactiver les actions non autorisées plutôt que crasher |
| Exec/Shell complexité PTY | Moyen | Moyen | Phase 2. Approche 1 (oc exec) en premier, fallback remotecommand ensuite |

---

## 12. Hors scope

Ce que l'outil ne fera **pas** :

- **Login** : pas de `oc login` intégré. L'utilisateur se connecte avant via `oc login` ou `kubectl`
- **Création de ressources** : pas de `create deployment`, `create service`. L'outil est orienté consultation + actions rapides
- **Édition YAML inline** : on ouvre `$EDITOR`, on ne réinvente pas un éditeur de texte
- **Monitoring avancé** : pas de graphiques, pas de Prometheus. Juste les métriques basiques CPU/Mem
- **Multi-cluster simultané** : un seul cluster actif à la fois (switch possible)
- **RBAC management** : pas de gestion des roles/rolebindings
- **Helm / Operators** : hors périmètre

/** @type {import('vls').VeturConfig} */
module.exports = {
  // override vscode settings
  // Notice: It only affects the settings used by Vetur.
  settings: {
    "vetur.useWorkspaceDependencies": true,
    "vetur.experimental.templateInterpolationService": false
  },
  // support monorepos
  projects: [
    './cosmos-frontend'
  ]
}

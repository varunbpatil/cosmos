import axios from 'axios'

// Create an axios instance with a custom baseURL.
// This is useful during development when the backend API server is running on a different port.
// VUE_APP_API_ROOT is set in the .env.development file inside the project root.
// The environment variable must have the VUE_APP_* prefix.
// See https://stackoverflow.com/questions/47407564/change-the-default-base-url-for-axios
//     https://cli.vuejs.org/guide/mode-and-env.html#modes
//     https://forum.vuejs.org/t/accessing-axios-in-vuex-module/29414/3
//     https://github.com/axios/axios#custom-instance-defaults
const VueAxios = axios.create()
VueAxios.defaults.baseURL = process.env.VUE_APP_API_ROOT

export default VueAxios

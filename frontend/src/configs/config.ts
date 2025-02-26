const SERVER_URL = import.meta.env.VITE_SERVER_URL;
const API_PATH = import.meta.env.VITE_API_PATH

export const API_URL = `http://${SERVER_URL}${API_PATH}`
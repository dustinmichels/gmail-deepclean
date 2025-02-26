/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// Window with added event listener for OAuth callback
interface Window {
  opener?: Window
  // Additional properties/methods if needed
}

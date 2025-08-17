import { DefineComponent } from 'vue'

interface IconProps {
  name: string
  size?: number | string
  color?: string
  class?: string | object | Array<string | object>
}

declare const Icon: DefineComponent<IconProps>

export default Icon
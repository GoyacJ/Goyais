import homeIcon from '@/assets/icons/heroicons/24/outline/home.svg?raw'
import squares2x2Icon from '@/assets/icons/heroicons/24/outline/squares-2x2.svg?raw'
import commandLineIcon from '@/assets/icons/heroicons/24/outline/command-line.svg?raw'
import cubeIcon from '@/assets/icons/heroicons/24/outline/cube.svg?raw'
import puzzlePieceIcon from '@/assets/icons/heroicons/24/outline/puzzle-piece.svg?raw'
import signalIcon from '@/assets/icons/heroicons/24/outline/signal.svg?raw'
import cog6ToothIcon from '@/assets/icons/heroicons/24/outline/cog-6-tooth.svg?raw'
import arrowPathIcon from '@/assets/icons/heroicons/24/outline/arrow-path.svg?raw'
import cloudArrowUpIcon from '@/assets/icons/heroicons/24/outline/cloud-arrow-up.svg?raw'
import shieldExclamationIcon from '@/assets/icons/heroicons/24/outline/shield-exclamation.svg?raw'
import exclamationTriangleIcon from '@/assets/icons/heroicons/24/outline/exclamation-triangle.svg?raw'
import magnifyingGlassIcon from '@/assets/icons/heroicons/24/outline/magnifying-glass.svg?raw'

export type IconName =
  | 'home'
  | 'canvas'
  | 'commands'
  | 'assets'
  | 'plugins'
  | 'streams'
  | 'settings'
  | 'refresh'
  | 'upload'
  | 'forbidden'
  | 'error'
  | 'not-found'
  | 'search'

const registry: Record<IconName, string> = {
  home: homeIcon,
  canvas: squares2x2Icon,
  commands: commandLineIcon,
  assets: cubeIcon,
  plugins: puzzlePieceIcon,
  streams: signalIcon,
  settings: cog6ToothIcon,
  refresh: arrowPathIcon,
  upload: cloudArrowUpIcon,
  forbidden: shieldExclamationIcon,
  error: exclamationTriangleIcon,
  'not-found': magnifyingGlassIcon,
  search: magnifyingGlassIcon,
}

export function resolveIconSvg(name: IconName): string {
  return registry[name]
}

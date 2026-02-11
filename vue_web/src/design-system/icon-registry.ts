/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

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
import chevronDownIcon from '@/assets/icons/heroicons/24/outline/chevron-down.svg?raw'

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
  | 'chevron-down'

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
  'chevron-down': chevronDownIcon,
}

export function resolveIconSvg(name: IconName): string {
  return registry[name]
}

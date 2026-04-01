export interface AvatarData {
  grid: boolean[][];
  fgColor: string;
  bgColor: string;
  cellColors: number[][];
  corners: number[][];
  fgColors: string[];
  gridWidth: number;
  gridHeight: number;
  numColors: number;
  curves: boolean;
}

export interface AvatarOptions {
  size?: number;
  gridWidth?: number;
  gridHeight?: number;
  numColors?: number;
  curves?: boolean;
  padding?: number;
}

export interface DeriveOptions {
  gridWidth?: number;
  gridHeight?: number;
  numColors?: number;
  curves?: boolean;
}

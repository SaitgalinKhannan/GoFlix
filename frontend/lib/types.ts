export type TorrentStatusValue =
  | "metadata"
  | "checking"
  | "queued"
  | "downloading"
  | "seeding"
  | "completed"
  | "paused"
  | "error";

export interface FileInfo {
  path: string;
  size: number;
}

export interface SpeedLimits {
  download: number; // bytes/s, 0 = unlimited
  upload: number; // bytes/s, 0 = unlimited
}

// Matches the Go TorrentStatus JSON contract.
export interface TorrentStatus {
  id: string;
  infoHash: string;
  name: string;
  status: TorrentStatusValue;
  progress: number; // 0..1
  downloadSpeed: number; // bytes/s
  uploadSpeed: number; // bytes/s
  seeders: number; // currently connected peers with all pieces
  leechers: number; // currently connected peers still downloading
  eta: number; // seconds remaining (0 = done/unknown)
  size: number; // bytes
  downloaded: number; // bytes
  addedAt: string; // ISO 8601
  completedAt?: string;
  error?: string;
  files?: FileInfo[];
}

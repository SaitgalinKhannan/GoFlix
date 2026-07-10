export type TorrentStatusValue =
  | "metadata"
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

// Matches the Go TorrentStatus JSON contract.
export interface TorrentStatus {
  id: string;
  infoHash: string;
  name: string;
  status: TorrentStatusValue;
  progress: number; // 0..1
  downloadSpeed: number; // bytes/s
  uploadSpeed: number; // bytes/s
  size: number; // bytes
  downloaded: number; // bytes
  addedAt: string; // ISO 8601
  completedAt?: string;
  error?: string;
  files?: FileInfo[];
}

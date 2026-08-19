// Offline scan queue for the check-in terminal. The beach kiosk's connection drops
// often enough that losing a scan (and having to ask the guest to show their QR again)
// is a real annoyance — this persists each scan locally the instant it's taken and
// replays it once connectivity returns, via the idempotent /checkin/validate/batch
// endpoint (safe to resubmit: the backend dedupes by clientScanId).

const DB_NAME = "p5-checkin-offline";
const STORE = "pending-scans";

export interface PendingScan {
  token: string;
  clientScanId: string;
  deviceScannedAt: string;
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => {
      req.result.createObjectStore(STORE, { keyPath: "clientScanId" });
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

export async function enqueueScan(scan: PendingScan): Promise<void> {
  const db = await openDB();
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(STORE, "readwrite");
    tx.objectStore(STORE).put(scan);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
  db.close();
}

export async function listPendingScans(): Promise<PendingScan[]> {
  const db = await openDB();
  const result = await new Promise<PendingScan[]>((resolve, reject) => {
    const tx = db.transaction(STORE, "readonly");
    const req = tx.objectStore(STORE).getAll();
    req.onsuccess = () => resolve(req.result as PendingScan[]);
    req.onerror = () => reject(req.error);
  });
  db.close();
  return result;
}

export async function removeScans(clientScanIds: string[]): Promise<void> {
  if (!clientScanIds.length) return;
  const db = await openDB();
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(STORE, "readwrite");
    const store = tx.objectStore(STORE);
    for (const id of clientScanIds) store.delete(id);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
  db.close();
}

export async function countPendingScans(): Promise<number> {
  return (await listPendingScans()).length;
}

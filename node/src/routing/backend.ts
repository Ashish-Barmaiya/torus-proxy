export class BackendServer {
  public readonly host: string;
  public readonly port: number;

  private _isHealthy: boolean = true;
  private _activeConnections: number = 0;

  constructor(host: string, port: number) {
    this.host = host;
    this.port = port;
  }

  // Getters
  get isHealthy(): boolean {
    return this._isHealthy;
  }

  get activeConnections(): number {
    return this._activeConnections;
  }

  get url(): string {
    return `http://${this.host}:${this.port}`;
  }

  // State Mutators
  markDead(): void {
    this._isHealthy = false;
  }

  markAlive(): void {
    this._isHealthy = true;
  }

  incrementActiveConnections(): void {
    this._activeConnections++;
  }

  decrementActiveConnections(): void {
    if (this._activeConnections > 0) {
      this._activeConnections--;
    }
  }
}

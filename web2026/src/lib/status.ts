export type StatusGlobal = 'Y' | 'N' | 'M'

export const STATUS_CLASS: Record<StatusGlobal, string> = {
    Y: 'bg-blue-100 text-blue-700',
    M: 'bg-yellow-100 text-yellow-700',
    N: 'bg-red-100 text-red-700'
}
export function getStatusLabel(status: StatusGlobal): string {
    switch (status) {
        case 'Y':
            return 'ACTIVE';
        case 'N':
            return 'DEACTIVE';
        default:
            return 'MAINTENANCE';
    }
}
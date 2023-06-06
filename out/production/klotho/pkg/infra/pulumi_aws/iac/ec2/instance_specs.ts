export const AL2_x86_64 = 'AL2_x86_64'
export const AL2_x86_64_GPU = 'AL2_x86_64_GPU'
export const AL2_ARM_64 = 'AL2_ARM_64'

export const EKS_AMI_INSTANCE_PREFIX_MAP = new Map([
    [
        AL2_x86_64,
        [
            'c1',
            'c3',
            'c4',
            'c5a',
            'c5d',
            'c5n',
            'c6i',
            'd2',
            'i2',
            'i3',
            'i3en',
            'i4i',
            'inf1',
            'm1',
            'm2',
            'm3',
            'm4',
            'm5',
            'm5a',
            'm5ad',
            'm5d',
            'm5zn',
            'm6i',
            'r3',
            'r4',
            'r5',
            'r5a',
            'r5ad',
            'r5d',
            'r5n',
            'r6i',
            't1',
            't2',
            't3',
            't3a',
            'z1d',
        ],
    ],
    [AL2_x86_64_GPU, ['g2', 'g3', 'g4dn']],
    [AL2_ARM_64, ['c6g', 'c6gd', 'c6gn', 'm6g', 'm6gd', 'r6g', 'r6gd', 't4g']],
])

export const getAmiFromInstanceType = (instanceType: string): string => {
    const instancePrefix = instanceType.split('.')[0]
    for (const key of EKS_AMI_INSTANCE_PREFIX_MAP.keys()) {
        const validPrefixes = EKS_AMI_INSTANCE_PREFIX_MAP.get(key)
        if (validPrefixes?.includes(instancePrefix)) {
            return key
        }
    }
    return ''
}

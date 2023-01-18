import { regexpMatch, regexpNotMatch } from '../sanitizer'

export const cacheCluster = {
    clusterNameValidation() {
        return {
            minLength: 1,
            maxLength: 40,
            rules: [
                regexpMatch('', /[a-zA-Z0-9-]/, (s) => s.replace(/[^a-zA-Z0-9-]/g, '-')),
                {
                    description: 'Identifier must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-+$/, ''),
                },
                regexpNotMatch('Identifier must not contain consecutive hyphens', /--/, (s) =>
                    s.replace(/--+/g, '-')
                ),
                regexpMatch('Identifier must start with a letter', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
            ],
        }
    },
}

// subnetGroupNameValidation are not documented and are inferred from Elasticache's CacheSubnetGroupName
export const subnetGroup = {
    subnetGroupNameValidation() {
        return {
            minLength: 1,
            maxLength: 255,
            rules: [
                regexpMatch(
                    '',
                    /^[a-z\d-]+$/, // uppercase is technically valid, but AWS will convert the value to lowercase
                    (s) => s.toLocaleLowerCase().replace(/[^a-z\d-]/g, '-')
                ),
                {
                    description: 'Identifier must not end with a hyphen',
                    validate: (s) => !s.endsWith('-'),
                    fix: (s) => s.replace(/-+$/, ''),
                },
                regexpNotMatch('Identifier must not contain consecutive hyphens', /--/, (s) =>
                    s.replace(/--+/g, '-')
                ),
                regexpMatch('Identifier must start with a letter', /^[a-zA-Z]/, (s) =>
                    s.replace(/^[^a-zA-Z]+/, '')
                ),
            ],
        }
    },
}

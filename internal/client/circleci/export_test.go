package circleci

// This file is compiled only into the test binary, so what it exports is
// reachable from this package's tests without becoming part of the package's public
// API.

// ResetOrbRegistryCapabilities forgets which hosts are known to serve the V3
// orb routes. Capability answers are cached for the life of the process, so a
// test standing up a fresh fake on a fresh address has to clear them.
var ResetOrbRegistryCapabilities = resetV3OrbRoutes

// ResetUserIds forgets which account id every host and token resolved to.
// Account ids are memoised for the life of the process, so a test asserting
// how often /api/v2/me is asked has to clear them.
var ResetUserIds = resetUserIds

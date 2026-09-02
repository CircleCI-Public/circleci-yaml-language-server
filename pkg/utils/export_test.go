package utils

// This file is compiled only into the test binary, so what it exports is
// reachable from pkg/utils tests without becoming part of the package's public
// API.

// ResetOrbRegistryCapabilities forgets which hosts are known to serve the V3
// orb routes. Capability answers are cached for the life of the process, so a
// test standing up a fresh fake on a fresh address has to clear them.
var ResetOrbRegistryCapabilities = resetV3OrbRoutes

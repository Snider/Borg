<?php

declare(strict_types=1);

namespace Borg\STMF\Tests;

require_once __DIR__ . '/../src/FormField.php';
require_once __DIR__ . '/../src/FormData.php';
require_once __DIR__ . '/../src/KeyPair.php';
require_once __DIR__ . '/../src/DecryptionException.php';
require_once __DIR__ . '/../src/InvalidPayloadException.php';
require_once __DIR__ . '/../src/STMF.php';

use Borg\STMF\STMF;
use Borg\STMF\KeyPair;

/**
 * Interoperability test - decrypts payloads encrypted by Go
 */
class InteropTest
{
    private array $vectors;
    private int $passed = 0;
    private int $failed = 0;

    public function __construct(string $vectorsFile)
    {
        $json = file_get_contents($vectorsFile);
        $this->vectors = json_decode($json, true);
        if ($this->vectors === null) {
            throw new \RuntimeException("Failed to parse test vectors: " . json_last_error_msg());
        }
    }

    public function run(): bool
    {
        echo "Running STMF Interoperability Tests\n";
        echo "===================================\n\n";

        foreach ($this->vectors as $vector) {
            $this->runVector($vector);
        }

        echo "\n===================================\n";
        echo "Results: {$this->passed} passed, {$this->failed} failed\n";

        return $this->failed === 0;
    }

    private function runVector(array $vector): void
    {
        $name = $vector['name'];
        echo "Testing: {$name}... ";

        try {
            // Create STMF instance with private key
            $stmf = new STMF($vector['private_key']);

            // Decrypt the payload
            $formData = $stmf->decrypt($vector['encrypted_b64']);

            // Verify fields
            $expectedFields = $vector['expected_fields'] ?? [];
            foreach ($expectedFields as $key => $expectedValue) {
                $actualValue = $formData->get($key);
                if ($actualValue !== $expectedValue) {
                    throw new \RuntimeException(
                        "Field '{$key}': expected " . json_encode($expectedValue) .
                        ", got " . json_encode($actualValue)
                    );
                }
            }

            // Verify metadata if present
            $expectedMeta = $vector['expected_meta'] ?? [];
            if ($expectedMeta) {
                $actualMeta = $formData->getMetadata();
                foreach ($expectedMeta as $key => $expectedValue) {
                    $actualValue = $actualMeta[$key] ?? null;
                    if ($actualValue !== $expectedValue) {
                        throw new \RuntimeException(
                            "Metadata '{$key}': expected " . json_encode($expectedValue) .
                            ", got " . json_encode($actualValue)
                        );
                    }
                }
            }

            // Verify field count
            $expectedCount = count($expectedFields);
            $actualCount = count($formData->fields());
            if ($actualCount !== $expectedCount) {
                throw new \RuntimeException(
                    "Field count: expected {$expectedCount}, got {$actualCount}"
                );
            }

            echo "PASS\n";
            $this->passed++;

        } catch (\Exception $e) {
            echo "FAIL\n";
            echo "  Error: " . $e->getMessage() . "\n";
            $this->failed++;
        }
    }
}

// Additional standalone tests
class StandaloneTests
{
    public static function runAll(): bool
    {
        echo "\nRunning Standalone PHP Tests\n";
        echo "============================\n\n";

        $passed = 0;
        $failed = 0;

        // Test 1: KeyPair generation
        echo "Testing: KeyPair generation... ";
        try {
            $kp = KeyPair::generate();
            if (strlen($kp->getPublicKey()) !== 32) {
                throw new \RuntimeException("Public key wrong length");
            }
            if (strlen($kp->getPrivateKey()) !== 32) {
                throw new \RuntimeException("Private key wrong length");
            }
            echo "PASS\n";
            $passed++;
        } catch (\Exception $e) {
            echo "FAIL: " . $e->getMessage() . "\n";
            $failed++;
        }

        // Test 2: KeyPair from private key
        echo "Testing: KeyPair from private key... ";
        try {
            $kp1 = KeyPair::generate();
            $kp2 = KeyPair::fromPrivateKeyBase64($kp1->getPrivateKeyBase64());
            if ($kp1->getPublicKeyBase64() !== $kp2->getPublicKeyBase64()) {
                throw new \RuntimeException("Public keys don't match");
            }
            echo "PASS\n";
            $passed++;
        } catch (\Exception $e) {
            echo "FAIL: " . $e->getMessage() . "\n";
            $failed++;
        }

        // Test 3: Invalid payload validation
        echo "Testing: Invalid payload detection... ";
        try {
            $kp = KeyPair::generate();
            $stmf = STMF::fromKeyPair($kp);
            $isValid = $stmf->validate("not-valid-base64!!!");
            if ($isValid) {
                throw new \RuntimeException("Should have rejected invalid payload");
            }
            $isValid2 = $stmf->validate(base64_encode("FAKE" . str_repeat("\x00", 100)));
            if ($isValid2) {
                throw new \RuntimeException("Should have rejected fake STMF");
            }
            echo "PASS\n";
            $passed++;
        } catch (\Exception $e) {
            echo "FAIL: " . $e->getMessage() . "\n";
            $failed++;
        }

        // Test 4: FormData methods
        echo "Testing: FormData methods... ";
        try {
            $fields = [
                \Borg\STMF\FormField::fromArray(['name' => 'email', 'value' => 'test@test.com']),
                \Borg\STMF\FormField::fromArray(['name' => 'tag', 'value' => 'one']),
                \Borg\STMF\FormField::fromArray(['name' => 'tag', 'value' => 'two']),
            ];
            $fd = new \Borg\STMF\FormData($fields, ['origin' => 'https://example.com']);

            if ($fd->get('email') !== 'test@test.com') {
                throw new \RuntimeException("get() failed");
            }
            if (!$fd->has('email')) {
                throw new \RuntimeException("has() failed");
            }
            if ($fd->has('nonexistent')) {
                throw new \RuntimeException("has() false positive");
            }

            $tags = $fd->getAll('tag');
            if (count($tags) !== 2 || $tags[0] !== 'one' || $tags[1] !== 'two') {
                throw new \RuntimeException("getAll() failed");
            }

            if ($fd->getOrigin() !== 'https://example.com') {
                throw new \RuntimeException("getOrigin() failed");
            }

            echo "PASS\n";
            $passed++;
        } catch (\Exception $e) {
            echo "FAIL: " . $e->getMessage() . "\n";
            $failed++;
        }

        echo "\n============================\n";
        echo "Standalone: {$passed} passed, {$failed} failed\n";

        return $failed === 0;
    }
}

// Run tests
if (php_sapi_name() === 'cli') {
    $vectorsFile = __DIR__ . '/test_vectors.json';

    if (!file_exists($vectorsFile)) {
        echo "Error: test_vectors.json not found.\n";
        echo "Generate it with: go run tests/generate_test_vectors.go > tests/test_vectors.json\n";
        exit(1);
    }

    // Check sodium extension
    if (!extension_loaded('sodium')) {
        echo "Error: sodium extension not loaded.\n";
        echo "Enable it in php.ini or install php-sodium.\n";
        exit(1);
    }

    $interop = new InteropTest($vectorsFile);
    $interopPassed = $interop->run();

    $standalonePassed = StandaloneTests::runAll();

    exit(($interopPassed && $standalonePassed) ? 0 : 1);
}

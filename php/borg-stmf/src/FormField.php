<?php

declare(strict_types=1);

namespace Borg\STMF;

/**
 * Represents a single form field
 */
class FormField
{
    public string $name;
    public string $value;
    public ?string $type;
    public ?string $filename;
    public ?string $mimeType;

    public function __construct(
        string $name,
        string $value,
        ?string $type = null,
        ?string $filename = null,
        ?string $mimeType = null
    ) {
        $this->name = $name;
        $this->value = $value;
        $this->type = $type;
        $this->filename = $filename;
        $this->mimeType = $mimeType;
    }

    /**
     * Check if this is a file field
     */
    public function isFile(): bool
    {
        return $this->type === 'file';
    }

    /**
     * Get the file content decoded from base64
     */
    public function getFileContent(): ?string
    {
        if (!$this->isFile()) {
            return null;
        }
        return base64_decode($this->value, true) ?: null;
    }

    /**
     * Create from array
     */
    public static function fromArray(array $data): self
    {
        return new self(
            $data['name'] ?? '',
            $data['value'] ?? '',
            $data['type'] ?? null,
            $data['filename'] ?? null,
            $data['mime'] ?? null
        );
    }
}

<?php

declare(strict_types=1);

namespace Borg\STMF;

/**
 * Represents decrypted form data
 */
class FormData
{
    /** @var FormField[] */
    private array $fields;

    /** @var array<string, string> */
    private array $metadata;

    /**
     * @param FormField[] $fields
     * @param array<string, string> $metadata
     */
    public function __construct(array $fields, array $metadata = [])
    {
        $this->fields = $fields;
        $this->metadata = $metadata;
    }

    /**
     * Get a field value by name
     */
    public function get(string $name): ?string
    {
        foreach ($this->fields as $field) {
            if ($field->name === $name) {
                return $field->value;
            }
        }
        return null;
    }

    /**
     * Get a field object by name
     */
    public function getField(string $name): ?FormField
    {
        foreach ($this->fields as $field) {
            if ($field->name === $name) {
                return $field;
            }
        }
        return null;
    }

    /**
     * Get all values for a field name (for multi-select)
     *
     * @return string[]
     */
    public function getAll(string $name): array
    {
        $values = [];
        foreach ($this->fields as $field) {
            if ($field->name === $name) {
                $values[] = $field->value;
            }
        }
        return $values;
    }

    /**
     * Get all fields
     *
     * @return FormField[]
     */
    public function fields(): array
    {
        return $this->fields;
    }

    /**
     * Check if a field exists
     */
    public function has(string $name): bool
    {
        foreach ($this->fields as $field) {
            if ($field->name === $name) {
                return true;
            }
        }
        return false;
    }

    /**
     * Convert to associative array (last value wins for duplicates)
     *
     * @return array<string, string>
     */
    public function toArray(): array
    {
        $result = [];
        foreach ($this->fields as $field) {
            $result[$field->name] = $field->value;
        }
        return $result;
    }

    /**
     * Get metadata
     *
     * @return array<string, string>
     */
    public function getMetadata(): array
    {
        return $this->metadata;
    }

    /**
     * Get a specific metadata value
     */
    public function getMeta(string $key): ?string
    {
        return $this->metadata[$key] ?? null;
    }

    /**
     * Get the origin (if set in metadata)
     */
    public function getOrigin(): ?string
    {
        return $this->metadata['origin'] ?? null;
    }

    /**
     * Get the timestamp (if set in metadata)
     */
    public function getTimestamp(): ?int
    {
        $ts = $this->metadata['timestamp'] ?? null;
        return $ts !== null ? (int) $ts : null;
    }

    /**
     * Create from decoded JSON array
     */
    public static function fromArray(array $data): self
    {
        $fields = [];
        foreach ($data['fields'] ?? [] as $fieldData) {
            $fields[] = FormField::fromArray($fieldData);
        }

        return new self($fields, $data['meta'] ?? []);
    }
}

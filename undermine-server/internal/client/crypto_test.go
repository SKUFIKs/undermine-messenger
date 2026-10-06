package client

import (
	"testing"
)

func TestE2EEEncryption(t *testing.T) {
	aliceKeys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Ошибка генерации ключей Алисы: %v", err)
	}

	bobKeys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Ошибка генерации ключей Боба: %v", err)
	}

	aliceShared, err := ComputeSharedKey(aliceKeys.PrivateKey, bobKeys.PublicKey)
	if err != nil {
		t.Fatalf("Ошибка вычисления секрета Алисой: %v", err)
	}

	bobShared, err := ComputeSharedKey(bobKeys.PrivateKey, aliceKeys.PublicKey)
	if err != nil {
		t.Fatalf("Ошибка вычисления секрета Бобом: %v", err)
	}

	if aliceShared != bobShared {
		t.Fatalf("Секреты не совпали!")
	}

	originalText := "Привет, Боб! Это сверхсекретное сквозное шифрование 🔒"
	nonce, ciphertext, err := Encrypt(originalText, aliceShared)
	if err != nil {
		t.Fatalf("Ошибка шифрования: %v", err)
	}

	decryptedText, err := Decrypt(nonce, ciphertext, bobShared)
	if err != nil {
		t.Fatalf("Ошибка расшифровки: %v", err)
	}

	if decryptedText != originalText {
		t.Fatalf("Текст не совпал! Получено: %s, ожидалось: %s", decryptedText, originalText)
	}

	t.Logf("Тест E2EE успешно пройден! Расшифрованный текст: %s", decryptedText)
}

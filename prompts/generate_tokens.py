#!/usr/bin/env python3
"""
Universal token generator for testing API limits.

Usage:
    python3 generate_tokens.py 50        # ~50 tokens
    python3 generate_tokens.py 50000     # ~50k tokens
    python3 generate_tokens.py 200000    # ~200k tokens
    python3 generate_tokens.py 500000    # ~500k tokens
"""

import random
import sys

paragraphs = [
    """Искусственный интеллект представляет собой область компьютерных наук, которая занимается созданием систем, способных выполнять задачи, требующие человеческого интеллекта. Это включает машинное обучение, обработку естественного языка, компьютерное зрение и робототехнику. Современные модели ИИ основаны на нейронных сетях с миллиардами параметров.""",

    """The development of large language models has revolutionized natural language processing. These models are trained on vast amounts of text data and can generate human-like responses, translate languages, write code, and answer complex questions. The architecture typically uses transformer networks with attention mechanisms.""",

    """Программирование на языке Go отличается простотой и эффективностью. Go был разработан в Google для создания надежных и масштабируемых систем. Ключевые особенности включают горутины для конкурентности, сборку мусора, статическую типизацию и быструю компиляцию.""",

    """REST API является архитектурным стилем для создания веб-сервисов. Основные принципы включают использование HTTP методов (GET, POST, PUT, DELETE), статусных кодов, и представление ресурсов в формате JSON или XML.""",

    """База данных PostgreSQL является мощной объектно-реляционной системой управления базами данных с открытым исходным кодом. Она поддерживает сложные запросы, транзакции ACID, индексы различных типов и полнотекстовый поиск.""",

    """Docker контейнеризация позволяет упаковывать приложения вместе со всеми зависимостями в изолированные контейнеры. Это обеспечивает консистентность между средами разработки и продакшена.""",

    """Машинное обучение разделяется на три основных типа: обучение с учителем, обучение без учителя и обучение с подкреплением. В обучении с учителем модель обучается на размеченных данных.""",

    """Cybersecurity encompasses the practices and technologies designed to protect systems, networks, and data from digital attacks. Key areas include network security, application security, and endpoint protection.""",

    """Микросервисная архитектура разбивает приложение на небольшие независимые сервисы, каждый из которых выполняет одну бизнес-функцию. Сервисы общаются через API или очереди сообщений.""",

    """Git является распределенной системой контроля версий, которая отслеживает изменения в файлах. Основные концепции включают коммиты, ветки, слияния и удаленные репозитории.""",
]

code_examples = [
    '''```go
func main() {
    http.HandleFunc("/api/users", handleUsers)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```''',
    '''```python
def process(data: list) -> dict:
    return {"count": len(data), "items": data}
```''',
    '''```sql
SELECT name, COUNT(*) as cnt FROM users GROUP BY name;
```''',
]

def generate_text(target_tokens: int) -> str:
    """Generate text with approximately target_tokens tokens."""
    target_chars = target_tokens * 4  # ~4 chars per token

    if target_tokens <= 100:
        # For small prompts, just use a single paragraph
        output = random.choice(paragraphs)
        while len(output) < target_chars:
            output += " " + random.choice(paragraphs)
        return output[:target_chars]

    output = "# Technical Documentation\n\n"
    section_num = 1

    while len(output) < target_chars:
        output += f"\n## Section {section_num}\n\n"

        for i in range(5):
            output += random.choice(paragraphs) + "\n\n"
            if i % 2 == 0 and len(output) < target_chars:
                output += random.choice(code_examples) + "\n\n"

        section_num += 1

    return output

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 generate_tokens.py <token_count>", file=sys.stderr)
        print("Example: python3 generate_tokens.py 50000", file=sys.stderr)
        sys.exit(1)

    try:
        target_tokens = int(sys.argv[1])
    except ValueError:
        print(f"Error: '{sys.argv[1]}' is not a valid number", file=sys.stderr)
        sys.exit(1)

    output = generate_text(target_tokens)
    print(output)

    char_count = len(output)
    estimated_tokens = char_count // 4
    print(f"\n---\nTarget: {target_tokens} tokens | Actual: ~{estimated_tokens} tokens ({char_count} chars)", file=sys.stderr)

if __name__ == "__main__":
    main()

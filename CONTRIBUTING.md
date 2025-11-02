# Contributing to StackMonitor PoC

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## 🚀 Getting Started

1. **Fork the repository**
2. **Clone your fork**
   ```bash
   git clone https://github.com/yourusername/stackmonitor-poc.git
   cd stackmonitor-poc
   ```
3. **Set up the development environment**
   ```bash
   docker compose up -d
   ```

## 📝 Development Workflow

### Making Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write clean, well-commented code
   - Follow existing code style
   - Add tests if applicable

3. **Test your changes**
   ```bash
   docker compose build
   docker compose up -d
   # Test the functionality
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "Add: descriptive commit message"
   ```

### Commit Message Guidelines

- Use clear, descriptive commit messages
- Prefix with type: `Add:`, `Fix:`, `Update:`, `Refactor:`, `Docs:`
- Examples:
  - `Add: new error categorization for database timeouts`
  - `Fix: memory leak in log parser`
  - `Update: README with new API endpoints`

## 🧪 Testing

Before submitting a pull request:

1. **Build all services**
   ```bash
   docker compose build
   ```

2. **Start services and verify**
   ```bash
   docker compose up -d
   docker compose ps
   ```

3. **Test functionality**
   - UI should load at http://localhost:3000
   - API endpoints should respond
   - Logs should be generated and displayed

4. **Check logs for errors**
   ```bash
   docker compose logs | grep -i error
   ```

## 📋 Pull Request Process

1. **Update documentation** if needed
2. **Ensure all tests pass** (if applicable)
3. **Update README.md** if adding new features
4. **Create pull request** with:
   - Clear title and description
   - Link to any related issues
   - Screenshots if UI changes

## 🎨 Code Style

### Go
- Use `gofmt` for formatting
- Follow standard Go conventions
- Add comments for exported functions

### Python
- Follow PEP 8
- Use meaningful variable names
- Add docstrings for functions

### JavaScript/React
- Use ESLint rules
- Prefer functional components
- Use meaningful component names

## 🐛 Reporting Bugs

When reporting bugs, please include:

1. **Description** of the bug
2. **Steps to reproduce**
3. **Expected behavior**
4. **Actual behavior**
5. **Environment**:
   - OS and version
   - Docker version
   - Browser (if UI-related)

## 💡 Feature Requests

For feature requests:

1. Check if the feature already exists or is planned
2. Open an issue describing:
   - The feature
   - Use case
   - Proposed implementation (if you have ideas)

## 📚 Documentation

- Update README.md for user-facing changes
- Add code comments for complex logic
- Update API documentation if endpoints change

## ✅ Checklist Before Submitting

- [ ] Code follows style guidelines
- [ ] Tests pass (if applicable)
- [ ] Documentation updated
- [ ] Commit messages are clear
- [ ] No console.log/debug statements left
- [ ] No hardcoded secrets or credentials

## 🤝 Code Review

All submissions require review. Please:

- Be open to feedback
- Respond to comments promptly
- Make requested changes
- Be respectful and constructive

Thank you for contributing! 🎉


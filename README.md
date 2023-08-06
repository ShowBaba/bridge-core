# QueryBridge

QueryBridge is an open-source project that aims to provide an easy way for users to create endpoints for performing CRUD (Create, Read, Update, Delete) operations on any number of databases. The platform allows users to eliminate the need for a dedicated backend service by providing a simple and direct interface for managing database connections and creating custom endpoints.

![Project Logo](Logo.png) 

## User Journey

The user journey in QueryBridge is designed to be seamless and user-friendly. Here's a step-by-step overview of the user journey:

1. **Account Creation**: Users can sign up for an account on the QueryBridge dashboard, providing their email and a secure password.

2. **Application Creation**: After signing up, users can create multiple applications within their account. Each application represents a distinct set of database connections and endpoints.

3. **Database Connection**: Within an application, users can add multiple databases by providing the necessary database connection credentials (e.g., host, port, username, password). QueryBridge will then establish a connection with the databases, fetch schema, tables, and column information.

4. **Real-time Activity Logs**: Users can view real-time logs of database operations such as schema discovery, table fetches, and other interactions with the connected databases.

5. **Endpoint Creation**: Once the database connections are set up, users can create custom endpoints. They can select a schema, a table, and any number of columns from the connected databases. Additionally, users can choose one of the CRUD methods (GET, POST, DELETE, PUT, PATCH) for the endpoint.

6. **Custom Endpoint URL**: After defining the endpoint, QueryBridge generates a custom URL for the user. This URL can be used to perform the specified CRUD operation on the selected database. For example, if it is a POST endpoint, the user can use the URL to send a POST request, and the specified operation will be executed on the connected database.

7. **Private or Public Endpoints**: Users have the option to declare an endpoint as private or public. If an endpoint is private, the backend will require an API key for validation before executing the request. The user can set the API key when creating the application to secure their private endpoints.

## Features

- [x] **User Authentication**
- [x] **Application Management**
- [x] **Database Connection Setup**
- [x] **Real-time Activity Logging**
- [x] **Simple Custom Endpoint Creation**
- [ ] **Add Custom Javascript Code to Endpoint Creation**

## API Documentation

API documentation can be found [here](https://documenter.getpostman.com/view/11688875/2s9XxwxaHN)

## Frontend Repository

The frontend React Tailwind application for the QueryBridge dashboard can be found [here](https://github.com/ShowBaba/bridge-dashboard.git)

## Figma Design

The design for the QueryBridge dashboard is available on Figma [here](https://www.figma.com/file/25wQoBo8shPdxays5XiDSg/query-bridge?type=design&node-id=66%3A9321&mode=design&t=LXIppltNrnTZnHSm-1)

## Live URLs

- Backend Server: [https://bridge-core.onrender.com]
- Frontend Dashboard: [https://bridge-dashboard.onrender.com]

## Contributing

We welcome contributions to QueryBridge! Whether it's bug fixes, new features, or improvements to documentation, we value your input. To contribute, follow these steps:

1. Fork the repository to your GitHub account.
2. Clone your forked repository to your local machine.
3. Create a new branch for your changes: `git checkout -b feature/new-feature` or `fix/issue-description`.
4. Make your modifications and commit your changes: `git commit -m "Description of changes"`.
5. Push your changes to your GitHub repository: `git push origin feature/new-feature`.
6. Open a pull request against the `main` branch of this repository.
7. Ensure your pull request follows the project's coding guidelines and passes any automated tests.

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information on contributing guidelines and code formatting.

## Installation


## Usage


## License

QueryBridge is released under the [MIT License](LICENSE). See the `LICENSE` file for details.

---

Thank you for your interest in QueryBridge! We hope this application provides a seamless experience for managing database connections and creating custom endpoints. We welcome your feedback and contributions to improve the project. Happy coding!

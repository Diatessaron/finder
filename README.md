Tinder for films. Simple: it recommends you films and either you like or you don't.

Endpoints:

1. Registration:
POST https://cognito-idp.eu-north-1.amazonaws.com

Headers:
- Content-Type: application/x-amz-json-1.1
- X-Amz-Target: AWSCognitoIdentityProviderService.SignUp

Body:
```
{
    "ClientId": "175ib75ecdr2tr9adg7oe6o94k",
    "Password": password,
    "Username":username,
    "UserAttributes": [
        {
            "Name": "email", //"email" is a key, keep it
            "Value": email
        }
    ]
}
```

2. Confirmation code:
POST https://cognito-idp.eu-north-1.amazonaws.com

Headers:
- Content-Type: application/x-amz-json-1.1
- X-Amz-Target: AWSCognitoIdentityProviderService.ConfirmSignUp

Body:
```
{
    "ClientId": "175ib75ecdr2tr9adg7oe6o94k",
    "Username": username as email
    "ConfirmationCode": confirmationCode
}
```

3. SignIn:
POST https://cognito-idp.eu-north-1.amazonaws.com

Headers:
- Content-Type: application/x-amz-json-1.1
- X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth

Body:
```
{
    "ClientId": "175ib75ecdr2tr9adg7oe6o94k",
    "AuthFlow": "USER_PASSWORD_AUTH",
    "AuthParameters": {
        "USERNAME": username as email,
        "PASSWORD": password
    }
}
``` 

4. Refresh token
POST https://cognito-idp.eu-north-1.amazonaws.com

Headers:
- Content-Type: application/x-amz-json-1.1
- X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth

Body:
```
{
    "AuthFlow": "REFRESH_TOKEN_AUTH",
    "ClientId": "175ib75ecdr2tr9adg7oe6o94k",
    "AuthParameters": {
        "REFRESH_TOKEN": refreshToken acquired after SigningIn
    }
}

```

5. Get films
GET https://j5szh4ivo1.execute-api.eu-north-1.amazonaws.com/default/get-films

Headers:
authorization: Bearer ${yourAccessToken}

6. Update film
If you like or do not like recommended film.

GET https://54zfj2agze.execute-api.eu-north-1.amazonaws.com/default/update-user-films?method=unlike&film=The Perfect Man

Query params:
- method=unlike
- film=The Perfect Man

Headers:
authorization: Bearer ${yourAccessToken}

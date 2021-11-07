<?php

declare(strict_types=1);
/**
 * This file is part of hyperf-ext/mail.
 *
 * @link     https://github.com/hyperf-ext/mail
 * @contact  eric@zhu.email
 * @license  https://github.com/hyperf-ext/mail/blob/master/LICENSE
 */
return [
    /*
    |--------------------------------------------------------------------------
    | Default Mailer
    |--------------------------------------------------------------------------
    |
    | This option controls the default mailer that is used to send any email
    | messages sent by your application. Alternative mailers may be setup
    | and used as needed; however, this mailer will be used by default.
    |
    */

    'default' => env('MAIL_MAILER', 'smtp'),

    /*
    |--------------------------------------------------------------------------
    | Mailer Configurations
    |--------------------------------------------------------------------------
    |
    | Here you may configure all of the mailers used by your application plus
    | their respective settings. Several examples have been configured for
    | you and you are free to add your own as your application requires.
    |
    | Supports a variety of mail "transport" drivers to be used while
    | sending an e-mail. You will specify which one you are using for your
    | mailers below. You are free to add additional mailers as required.
    |
    */

    'mailers' => [
        'smtp' => [
            'transport' => \HyperfExt\Mail\Transport\SmtpTransport::class,
            'options' => [
                'host' => env('MAIL_SMTP_HOST', 'smtp.mailgun.org'),
                'port' => env('MAIL_SMTP_PORT', 587),
                'encryption' => env('MAIL_SMTP_ENCRYPTION', 'tls'),
                'username' => env('MAIL_SMTP_USERNAME'),
                'password' => env('MAIL_SMTP_PASSWORD'),
                'timeout' => env('MAIL_SMTP_TIMEOUT'),
                'auth_mode' => env('MAIL_SMTP_AUTH_MODE'),
            ],
        ],

        'aws_ses' => [
            'transport' => \HyperfExt\Mail\Transport\AwsSesTransport::class,
            'options' => [
                'credentials' => [
                    'key' => env('MAIL_AWS_SES_ACCESS_KEY_ID'),
                    'secret' => env('MAIL_AWS_SES_SECRET_ACCESS_KEY'),
                ],
                'region' => env('MAIL_AWS_SES_REGION'),
            ],
        ],

        'aliyun_dm' => [
            'transport' => \HyperfExt\Mail\Transport\AliyunDmTransport::class,
            'options' => [
                'access_key_id' => env('MAIL_ALIYUN_DM_ACCESS_KEY_ID'),
                'access_secret' => env('MAIL_ALIYUN_DM_ACCESS_SECRET'),
                'region_id' => env('MAIL_ALIYUN_DM_REGION_ID'),
                'click_trace' => env('MAIL_ALIYUN_DM_CLICK_TRACE', '0'),
            ],
        ],

        'mailgun' => [
            'transport' => \HyperfExt\Mail\Transport\MailgunTransport::class,
            'options' => [
                'domain' => env('MAIL_MAILGUN_DOMAIN'),
                'key' => env('MAIL_MAILGUN_KEY'),
                'endpoint' => env('MAIL_MAILGUN_ENDPOINT', 'api.mailgun.net'),
            ],
        ],

        'postmark' => [
            'transport' => \HyperfExt\Mail\Transport\PostmarkTransport::class,
            'options' => [
                'token' => env('MAIL_POSTMARK_TOKEN'),
            ],
        ],

        'sendmail' => [
            'transport' => \HyperfExt\Mail\Transport\SendmailTransport::class,
            'options' => [
                'path' => env('MAIL_SENDMAIL_PATH', '/usr/sbin/sendmail -bs'),
            ],
        ],

        'log' => [
            'transport' => \HyperfExt\Mail\Transport\LogTransport::class,
            'options' => [
                'name' => 'mail.local',
                'group' => 'default',
            ],
        ],
    ],

    /*
    |--------------------------------------------------------------------------
    | Global "From" Address
    |--------------------------------------------------------------------------
    |
    | You may wish for all e-mails sent by your application to be sent from
    | the same address. Here, you may specify a name and address that is
    | used globally for all e-mails that are sent by your application.
    |
    */

    'from' => [
        'address' => env('MAIL_FROM_ADDRESS', 'hello@example.com'),
        'name' => env('MAIL_FROM_NAME', 'Example'),
    ],

    /*
    |--------------------------------------------------------------------------
    | Logger Options
    |--------------------------------------------------------------------------
    |
    | The `hyperf/logger` component is required if enabled.
    */

    'logger' => [
        'enabled' => false,
        'name' => 'mail',
        'group' => 'default',
    ],
];

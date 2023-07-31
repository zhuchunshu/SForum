<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\SendService;

use App\Plugins\Mail\src\Annotation\SendService;
use App\Plugins\Mail\src\SendServiceInterface;
use PHPMailer\PHPMailer\PHPMailer;
#[SendService]
class Smtp implements SendServiceInterface
{
    public function get_service_name() : string
    {
        return 'SMTP发信';
    }
    public function view() : string
    {
        return 'Mail::config.smtp';
    }
    public function handler() : string
    {
        return \App\Plugins\Mail\src\Handler\Smtp::class;
    }
    public function send($email, $subject, $body)
    {
        $mail = new PHPMailer(true);
        try {
            $port = (int) get_options('MAIL_SMTP_PORT');
            //Server settings
            $mail->SMTPDebug = \PHPMailer\PHPMailer\SMTP::DEBUG_OFF;
            //Enable verbose debug output
            $mail->isSMTP();
            //Send using SMTP
            $mail->Host = get_options('MAIL_SMTP_HOST');
            //Set the SMTP server to send through
            $mail->SMTPAuth = true;
            //Enable SMTP authentication
            $mail->Username = get_options('MAIL_SMTP_USERNAME');
            //SMTP username
            $mail->Password = get_options('MAIL_SMTP_PASSWORD');
            //SMTP password
            $SMTPSecure = '';
            if ($port === 465) {
                $SMTPSecure = PHPMailer::ENCRYPTION_SMTPS;
            }
            if ($port === 587) {
                $SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;
            }
            $mail->SMTPSecure = $SMTPSecure;
            //Enable implicit TLS encryption
            $mail->Port = $port;
            //                                    //TCP port to connect to; use 587 if you have set `SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS`
            $mail->CharSet = 'UTF-8';
            //Recipients
            $mail->setFrom(get_options('MAIL_SMTP_FORM_MAIL'), get_options('MAIL_SMTP_FORM_NAME'));
            $mail->isHTML(true);
            //Set email format to HTML
            $mail->addAddress($email);
            $mail->Subject = $subject;
            $mail->Body = $body;
            return $mail->send();
        } catch (\Exception $e) {
            admin_log()->insert('mail', 'sendMail(SMTP发信)', '发信失败!', ['error' => $e]);
            throw new \RuntimeException($mail->ErrorInfo);
        }
    }
}
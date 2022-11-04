<?php


namespace App\Plugins\Mail\src;

use Exception;
use Illuminate\Support\Arr;
use PHPMailer\PHPMailer\PHPMailer;
use PHPMailer\PHPMailer\SMTP;

class Mail
{
    /**
     * SMTP 主机地址
     * @var string|array|bool|mixed|void
     */
    public string $host;

    /**
     * SMTP 用户名
     * @var string|array|bool|mixed|void
     */
    public string $username;

    /**
     * SMTP 密码
     * @var string|array|bool|mixed|void
     */
    public string $password;

    /**
     * SMTP 端口
     * @var int|array|bool|mixed|string|void
     */
    public int $port;

    /**
     * 发件人名称
     * @var string
     */
    public string $form_name;

    /**
     * 发件人邮箱
     * @var string
     */
    public string $form_email;

    /**
     * 编码
     * @var string
     */
    public string $CharSet ="UTF-8";

    public string $SMTPSecure = PHPMailer::ENCRYPTION_SMTPS;


    public function init(): Mail
    {
        $this->host = get_options("MAIL_SMTP_HOST");
        $this->username = get_options("MAIL_SMTP_USERNAME");
        $this->password = get_options("MAIL_SMTP_PASSWORD");
        $this->port = get_options("MAIL_SMTP_PORT");
        $this->form_name = get_options("MAIL_SMTP_FORM_NAME");
        $this->form_email = get_options("MAIL_SMTP_FORM_MAIL");
        if($this->port===465){
            $this->SMTPSecure = PHPMailer::ENCRYPTION_SMTPS;
        }
        if($this->port===587){
            $this->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;
        }
        if($this->port===25){
            $this->SMTPSecure = "";
        }
        return $this;
    }



    public function mail(): ?PHPMailer
    {
        $mail = new PHPMailer(false);
        try {
            //Server settings
            $mail->SMTPDebug = SMTP::DEBUG_OFF;                      //Enable verbose debug output
            $mail->isSMTP();                                            //Send using SMTP
            $mail->Host       = $this->host;                     //Set the SMTP server to send through
            $mail->SMTPAuth   = true;                                   //Enable SMTP authentication
            $mail->Username   = $this->username;                     //SMTP username
            $mail->Password   = $this->password;                               //SMTP password
            $mail->SMTPSecure = $this->SMTPSecure;            //Enable implicit TLS encryption
            $mail->Port       = $this->port; //                                    //TCP port to connect to; use 587 if you have set `SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS`
            $mail->CharSet = $this->CharSet;                     //

            //Recipients
            $mail->setFrom($this->form_email, $this->form_name);


            $mail->isHTML(true);                                  //Set email format to HTML

            return $mail;
        } catch (Exception $e) {
            admin_log()->insert('mail','sendMail(默认发信)','发信失败!',[
                'error' => $e
            ]);
            throw new \RuntimeException($mail->ErrorInfo);
        }
    }

    public function data(): Mail
    {
        return $this;
    }
}
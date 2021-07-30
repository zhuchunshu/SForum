<?php

use App\Plugins\Mail\src\Mail;

if(!function_exists("Email")){
    function Email(): ?\PHPMailer\PHPMailer\PHPMailer
    {
        return (new Mail())->init()->mail();
    }
}

if(!function_exists("EmailData")){
    function EmailData(): Mail
    {
        return (new Mail())->init()->data();
    }
}
<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateShortcodePaidPostOrder extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('shortcode_paid_post_order', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->bigInteger('post_id')->comment('帖子ID');
            // 用户id
            $table->bigInteger('user_id')->comment('用户ID');
            $table->datetimes();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('shortcode_paid_post_order');
    }
}

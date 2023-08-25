<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateShortcodePaidPost extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('shortcode_paid_post', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->bigInteger('post_id')->comment('帖子ID');
            $table->string('type')->default('money')->nullable();
            $table->bigInteger('amount');
            $table->datetimes();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('shortcode_paid_post');
    }
}

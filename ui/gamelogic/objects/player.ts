import Phaser from "phaser";
import { Movements, Plugins, Sounds, Spritesheets } from "../enums";
import { GameLogicPlugin } from "../plugins";
import { Character } from "./character";
import { Bottle } from "./bottle";

export class Player extends Character {
    private isWalking: boolean;
    private isJumping: boolean;
    private isIdling: boolean;
    private isDead: boolean;
    private speed: number;
    private jumpVelocity: number;
    private onCooldown = false;
    keys: any;
    state: Movements = Movements.Idle;
    gameLogic: GameLogicPlugin | null = null;
    timer: number = 0;

    constructor(scene: Phaser.Scene, x: number, y: number, speed=50, jumpVelocity=150, tint: number | null = null) {
        super(scene, x, y, tint);

        if (tint !== null)
            this.setTint(tint)
        this.speed = speed;
        this.jumpVelocity = jumpVelocity;
        this.isWalking = false;
        this.isJumping = false;
        this.isIdling = true;

        this.gameLogic = this.scene.plugins.get(Plugins.GameLogic) as GameLogicPlugin;

        this.isDead = false;

        const { LEFT, RIGHT, UP, W, A, D } = Phaser.Input.Keyboard.KeyCodes;
        this.keys = scene.input?.keyboard?.addKeys({
            left: LEFT,
            right: RIGHT,
            up: UP,
            space: Phaser.Input.Keyboard.KeyCodes.SPACE
        });
    }

    walkRight() {
        if (!this.isDead) {
            this.checkJump();
            this.setVelocityX(this.speed);
            this.setFlipX(false);
            this.isIdling = false;
            if (!this.isWalking && !this.isJumping) {
                this.state = Movements.Walk;
                this.play(Movements.Walk);
                this.isWalking = true;
            };
        }
    }

    walkLeft() {
        if (!this.isDead) {
            this.checkJump();
            this.setVelocityX(-1*this.speed);
            this.setFlipX(true);
            this.isIdling = false;
            if (!this.isWalking && !this.isJumping) {
                this.state = Movements.Walk;
                this.play(Movements.Walk);
                this.isWalking = true;
            }
        }
    }

    jump() {
        if (this.canJump && !this.isDead) {
            this.setVelocityY(-1*this.jumpVelocity);
            this.state = Movements.Jump;
            this.play(Movements.Jump);
            this.isJumping = true;
            this.isIdling = false;
            this.isWalking = false;
            this.scene.sound.play(Sounds.Jump);
        }
    }

    idle() {
        this.checkJump();
        if (!this.isJumping && !this.isIdling && !this.isDead) {
            this.state = Movements.Idle;
            this.isIdling = true;
            this.isWalking = false;
            this.play(Movements.Idle);
        }
    }

    startCooldown() {
        this.onCooldown = true;
        setTimeout(() => {
            this.onCooldown = false;
        }, 1000);
    }

    update(time: number, delta: number) {
        this.setVelocityX(0);
        if (this.keys.right.isDown) {
            this.walkRight();
        } else if (this.keys.left.isDown) {
            this.walkLeft();
        } else {
            this.idle();
        }
        if ((this.keys.up.isDown || this.keys.space.isDown)) {
            this.jump();
        }

        this.timer += delta;
        if (this.timer > 100) {
            this.timer = 0;
            this.gameLogic?.sendPosition(this.x, this.y, this.state);
        }
    }

    private checkJump() {
        if (this.canJump) {
            this.isJumping = false;
        }
    }

    private get canJump() {
        return this.body?.blocked.down;
    }
}